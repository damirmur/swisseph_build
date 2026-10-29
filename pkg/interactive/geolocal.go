package interactive

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/damirmur/swisseph_build/pkg/geo"
)

// Локальный геокодер: координаты и IANA-таймзона берутся из индекса GeoNames
// (cmd/geobuild → geo/cities.idx). Сеть при этом не нужна, поэтому расчёт по
// городу не зависит от лимитов Nominatim и от доступности timeapi.io.

var (
	geoOnce   sync.Once
	geoIdx    *geo.Index
	geoIdxErr error
)

// geoIndexPaths — где искать индекс: переменная окружения, каталог рядом с
// бинарём, общий путь сборки.
func geoIndexPaths() []string {
	var out []string
	if p := strings.TrimSpace(os.Getenv("GEO_INDEX_PATH")); p != "" {
		out = append(out, p)
	}
	if exe, err := os.Executable(); err == nil {
		out = append(out, filepath.Join(filepath.Dir(exe), "geo", "cities.idx"))
	}
	out = append(out,
		"/home/dmyr/swisseph_build/geo/cities.idx",
		"geo/cities.idx",
	)
	return out
}

func localIndex() *geo.Index {
	geoOnce.Do(func() {
		for _, p := range geoIndexPaths() {
			if st, err := os.Stat(p); err != nil || st.IsDir() {
				continue
			}
			ix, err := geo.Open(p)
			if err != nil {
				geoIdxErr = err
				continue
			}
			geoIdx = ix
			return
		}
	})
	return geoIdx
}

// LocalGeoAvailable сообщает, загружен ли локальный индекс (для диагностики).
func LocalGeoAvailable() bool { return localIndex() != nil }

// LocalGeoError возвращает причину, по которой индекс не загрузился.
func LocalGeoError() error { return geoIdxErr }

// localGeoData ищет город в локальном индексе.
func localGeoData(input *UserInput) (*GeoData, bool) {
	ix := localIndex()
	if ix == nil || strings.TrimSpace(input.City) == "" {
		return nil, false
	}
	res, ok := ix.Lookup(input.City)
	if !ok {
		return nil, false
	}
	return &GeoData{
		City:      res.Place.Name,
		Latitude:  res.Place.Lat,
		Longitude: res.Place.Lon,
		Timezone:  res.Place.Timezone,
		UTCOffset: tzOffsetAt(res.Place.Timezone, input.Date, input.Time),
	}, true
}

// tzOffsetAt считает смещение зоны на конкретную дату и время (с историческими
// правилами перехода на летнее время), а не «сейчас»: иначе карта человека,
// родившегося летом при действовавшем тогда DST, уезжает на час.
func tzOffsetAt(tz, dateStr, timeStr string) float64 {
	loc, err := time.LoadLocation(strings.TrimSpace(tz))
	if err != nil {
		return 0
	}
	d := strings.TrimSpace(dateStr)
	if d == "" {
		d = time.Now().Format("2006-01-02")
	}
	tm := strings.TrimSpace(timeStr)
	if tm == "" {
		tm = "12:00"
	}
	parsed, err := time.ParseInLocation("2006-01-02 15:04", d+" "+tm, loc)
	if err != nil {
		// Дата могла прийти в «человеческом» виде (15.05.1990, 15/05/1990).
		if dd, derr := parseDate(d); derr == nil {
			if tt, terr := parseTime(tm); terr == nil {
				parsed, err = time.ParseInLocation("2006-01-02 15:04", dd+" "+tt, loc)
			}
		}
		if err != nil {
			return 0
		}
	}
	_, offset := parsed.Zone()
	return float64(offset) / 3600
}

// ─── кэш сетевых ответов (фолбэк для мест вне дампа) ────────────────────────

type netGeoEntry struct {
	gd *GeoData
	at time.Time
}

var (
	netGeoMu    sync.Mutex
	netGeoCache = map[string]netGeoEntry{}
)

const netGeoTTL = 30 * 24 * time.Hour

// netGeoCached возвращает координаты, найденные сетью ранее (ключ — имя города).
func netGeoCached(city string) (*GeoData, bool) {
	key := strings.ToLower(strings.TrimSpace(city))
	if key == "" {
		return nil, false
	}
	netGeoMu.Lock()
	defer netGeoMu.Unlock()
	e, ok := netGeoCache[key]
	if !ok || time.Since(e.at) > netGeoTTL {
		return nil, false
	}
	cp := *e.gd
	return &cp, true
}

func netGeoStore(city string, gd *GeoData) {
	key := strings.ToLower(strings.TrimSpace(city))
	if key == "" || gd == nil {
		return
	}
	netGeoMu.Lock()
	defer netGeoMu.Unlock()
	cp := *gd
	netGeoCache[key] = netGeoEntry{gd: &cp, at: time.Now()}
}

// nameMatchesCity проверяет, что в названии найденного объекта действительно есть
// введённое название (Nominatim ищет нечётко и легко отдаёт чужой город).
// Сравнение — по нормализованным строкам, по границам слов и запятых.
func nameMatchesCity(query, displayName string) bool {
	q := geo.Normalize(query)
	if q == "" {
		return false
	}
	d := geo.Normalize(displayName)
	if d == "" {
		return false
	}
	if d == q {
		return true
	}
	for _, part := range strings.FieldsFunc(d, func(r rune) bool { return r == ',' || r == ';' || r == '(' }) {
		if strings.TrimSpace(part) == q {
			return true
		}
	}
	return strings.Contains(d, " "+q+" ") || strings.HasPrefix(d, q+" ") || strings.HasSuffix(d, " "+q)
}
