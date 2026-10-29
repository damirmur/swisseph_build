// Команда geobuild собирает локальный индекс геокодера из дампа GeoNames.
//
// Загрузка и сборка (раз в месяц достаточно):
//
//	curl -O https://download.geonames.org/export/dump/cities1000.zip
//	go run ./cmd/geobuild -zip cities1000.zip -out geo/cities.idx
//
// Формат входа — cities1000.txt из архива: tab-separated, 19 колонок
// (geonameid, name, asciiname, alternatenames, lat, lon, fclass, fcode,
//
//	country, cc2, admin1..4, population, elevation, dem, timezone, modified).
//
// Лицензия данных GeoNames — CC BY 4.0: на сайте нужна ссылка-атрибуция.
package main

import (
	"archive/zip"
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/damirmur/swisseph_build/pkg/geo"
)

func main() {
	in := flag.String("in", "", "cities1000.txt из дампа GeoNames")
	zipPath := flag.String("zip", "", "cities1000.zip (вместо -in)")
	out := flag.String("out", "geo/cities.idx", "куда писать индекс")
	limit := flag.Int("limit", 0, "ограничить число записей (для отладки)")
	flag.Parse()

	if *in == "" && *zipPath == "" {
		log.Fatal("укажите -in cities1000.txt или -zip cities1000.zip")
	}

	rc, err := openSource(*in, *zipPath)
	if err != nil {
		log.Fatalf("чтение дампа: %v", err)
	}
	defer rc.Close()

	places, skipped, err := parse(rc, *limit)
	if err != nil {
		log.Fatalf("разбор дампа: %v", err)
	}
	if len(places) == 0 {
		log.Fatal("в дампе не нашлось ни одного населённого пункта")
	}

	if err := geo.Build(places, *out); err != nil {
		log.Fatalf("запись индекса: %v", err)
	}
	st, err := os.Stat(*out)
	if err != nil {
		log.Fatalf("проверка индекса: %v", err)
	}
	fmt.Printf("готово: %s — %d мест, пропущено %d строк, %.1f МБ\n",
		*out, len(places), skipped, float64(st.Size())/(1<<20))

	// Самопроверка: индекс должен открываться и находить Москву.
	ix, err := geo.Open(*out)
	if err != nil {
		log.Fatalf("индекс не открывается: %v", err)
	}
	for _, probe := range []string{"Москва", "Moscow", "Новосибирск"} {
		if res, ok := ix.Lookup(probe); ok {
			fmt.Printf("  %s → %s (%s, %.4f, %.4f, %.0f чел.)\n",
				probe, res.Place.Name, res.Place.Timezone, res.Place.Lat, res.Place.Lon, float64(res.Place.Population))
		} else {
			fmt.Printf("  %s → не найдено\n", probe)
		}
	}
}

// openSource открывает либо zip, либо распакованный txt.
func openSource(txt, zipPath string) (io.ReadCloser, error) {
	if zipPath != "" {
		zr, err := zip.OpenReader(zipPath)
		if err != nil {
			return nil, err
		}
		for _, f := range zr.File {
			if strings.HasSuffix(f.Name, ".txt") {
				rc, err := f.Open()
				if err != nil {
					return nil, err
				}
				return &zipEntry{rc: rc, zr: zr}, nil
			}
		}
		return nil, fmt.Errorf("в архиве нет .txt")
	}
	return os.Open(txt)
}

type zipEntry struct {
	rc io.ReadCloser
	zr *zip.ReadCloser
}

func (z *zipEntry) Read(p []byte) (int, error) { return z.rc.Read(p) }
func (z *zipEntry) Close() error {
	err := z.rc.Close()
	if e := z.zr.Close(); err == nil {
		err = e
	}
	return err
}

// parse разбирает TSV GeoNames в набор мест для индекса.
func parse(r io.Reader, limit int) ([]geo.SourcePlace, int, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)

	var out []geo.SourcePlace
	skipped := 0
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) < 19 {
			skipped++
			continue
		}
		tz := strings.TrimSpace(f[17])
		if tz == "" {
			skipped++ // без часового пояса место бесполезно: карта будет неверной
			continue
		}
		id, err := strconv.ParseInt(f[0], 10, 32)
		if err != nil || id <= 0 {
			skipped++
			continue
		}
		lat, err1 := strconv.ParseFloat(f[4], 64)
		lon, err2 := strconv.ParseFloat(f[5], 64)
		if err1 != nil || err2 != nil {
			skipped++
			continue
		}
		pop, _ := strconv.ParseInt(strings.TrimSpace(f[14]), 10, 64)

		names := make([]string, 0, 8)
		names = append(names, f[1])
		if f[2] != "" && f[2] != f[1] {
			names = append(names, f[2])
		}
		if f[3] != "" {
			for _, alt := range strings.Split(f[3], ",") {
				alt = strings.TrimSpace(alt)
				if alt != "" {
					names = append(names, alt)
				}
			}
		}

		out = append(out, geo.SourcePlace{
			ID:       int32(id),
			Name:     f[1],
			Lat:      lat,
			Lon:      lon,
			Pop:      pop,
			Timezone: tz,
			Country:  strings.ToUpper(f[8]),
			Names:    names,
		})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, skipped, sc.Err()
}
