package geo

import (
	"os"
	"path/filepath"
	"testing"
)

func samplePlaces() []SourcePlace {
	return []SourcePlace{
		{
			ID: 524901, Name: "Moscow", Lat: 55.7522, Lon: 37.6156, Pop: 10381222,
			Timezone: "Europe/Moscow", Country: "RU",
			Names: []string{"Moscow", "Moskva", "Москва", "Москов", "Moskau"},
		},
		{
			// Тёзка: Moscow, Idaho — не должен выигрывать у российской.
			ID: 5601538, Name: "Moscow", Lat: 46.7324, Lon: -117.0002, Pop: 25060,
			Timezone: "America/Los_Angeles", Country: "US",
			Names: []string{"Moscow", "Hog Heaven"},
		},
		{
			ID: 498677, Name: "Novosibirsk", Lat: 55.0415, Lon: 82.9346, Pop: 1612833,
			Timezone: "Asia/Novosibirsk", Country: "RU",
			Names: []string{"Novosibirsk", "Новосибирск"},
		},
		{
			ID: 498817, Name: "Korolyov", Lat: 55.9142, Lon: 37.8256, Pop: 183402,
			Timezone: "Europe/Moscow", Country: "RU",
			Names: []string{"Korolyov", "Королёв", "Королев"},
		},
		{
			ID: 3448439, Name: "São Paulo", Lat: -23.5475, Lon: -46.6361, Pop: 10021295,
			Timezone: "America/Sao_Paulo", Country: "BR",
			Names: []string{"São Paulo", "Sao Paulo", "Сан-Паулу"},
		},
	}
}

func buildTestIndex(t *testing.T) *Index {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cities.idx")
	if err := Build(samplePlaces(), path); err != nil {
		t.Fatalf("сборка индекса: %v", err)
	}
	ix, err := Open(path)
	if err != nil {
		t.Fatalf("открытие индекса: %v", err)
	}
	return ix
}

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"Москва":        "москва",
		"  МОСКВА  ":    "москва",
		"Королёв":       "королев",
		"São Paulo":     "sao paulo",
		"New   York":    "new york",
		"Saint-Étienne": "saint-etienne",
		"":              "",
		"   ":           "",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q)=%q, ожидалось %q", in, got, want)
		}
	}
}

func TestLookupRanksByPopulation(t *testing.T) {
	ix := buildTestIndex(t)

	for _, q := range []string{"Москва", "москва", "Moscow", "Moskva"} {
		res, ok := ix.Lookup(q)
		if !ok {
			t.Fatalf("Lookup(%q): не найдено", q)
		}
		if res.Place.ID != 524901 {
			t.Errorf("Lookup(%q) → %s (%s), ожидалась Москва (RU)", q, res.Place.Name, res.Place.Country)
		}
		if res.Place.Timezone != "Europe/Moscow" {
			t.Errorf("Lookup(%q): timezone=%q", q, res.Place.Timezone)
		}
		// «Moscow» есть и в США (Idaho), кириллическая «Москва» — только в РФ.
		wantMatches := 1
		if Normalize(q) == "moscow" {
			wantMatches = 2
		}
		if res.Matches != wantMatches {
			t.Errorf("Lookup(%q): совпадений %d, ожидалось %d", q, res.Matches, wantMatches)
		}
	}
}

func TestLookupAlternateNames(t *testing.T) {
	ix := buildTestIndex(t)
	cases := []struct {
		q       string
		id      int32
		tz      string
		country string
	}{
		{"Новосибирск", 498677, "Asia/Novosibirsk", "RU"},
		{"Королёв", 498817, "Europe/Moscow", "RU"},
		{"Королев", 498817, "Europe/Moscow", "RU"},
		{"São Paulo", 3448439, "America/Sao_Paulo", "BR"},
		{"Sao Paulo", 3448439, "America/Sao_Paulo", "BR"},
		{"Сан-Паулу", 3448439, "America/Sao_Paulo", "BR"},
	}
	for _, c := range cases {
		res, ok := ix.Lookup(c.q)
		if !ok {
			t.Errorf("Lookup(%q): не найдено", c.q)
			continue
		}
		if res.Place.ID != c.id || res.Place.Timezone != c.tz || res.Place.Country != c.country {
			t.Errorf("Lookup(%q) → %d %s %s, ожидалось %d %s %s",
				c.q, res.Place.ID, res.Place.Timezone, res.Place.Country, c.id, c.tz, c.country)
		}
	}
}

func TestLookupUnknown(t *testing.T) {
	ix := buildTestIndex(t)
	for _, q := range []string{"Ззззъъъ", "qwertyuiopasdf", "", "   "} {
		if res, ok := ix.Lookup(q); ok {
			t.Errorf("Lookup(%q) неожиданно нашёл %s (%s)", q, res.Place.Name, res.Place.Country)
		}
	}
}

func TestBuildStats(t *testing.T) {
	ix := buildTestIndex(t)
	if got := ix.Places(); got != len(samplePlaces()) {
		t.Errorf("мест в индексе %d, ожидалось %d", got, len(samplePlaces()))
	}
	// ключи: уникальные нормализованные имена (Королёв/Королев — один ключ)
	if ix.Keys() < 10 || ix.Keys() > 20 {
		t.Errorf("ключей %d — похоже на ошибку сборки", ix.Keys())
	}
}

func TestOpenBrokenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.idx")
	if err := os.WriteFile(path, []byte("не индекс"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(path); err == nil {
		t.Fatal("битый файл открылся без ошибки")
	}
	if _, err := Open(filepath.Join(t.TempDir(), "nope.idx")); err == nil {
		t.Fatal("отсутствующий файл открылся без ошибки")
	}
}

// TestRealIndex — проверка на настоящем дампе, если он собран (geo/cities.idx).
// Пропускается, когда индекса нет: тест не должен требовать докачки данных.
func TestRealIndex(t *testing.T) {
	path := os.Getenv("GEO_INDEX_PATH")
	if path == "" {
		path = filepath.Join("..", "..", "geo", "cities.idx")
	}
	if _, err := os.Stat(path); err != nil {
		t.Skipf("нет собранного индекса (%s) — пропуск", path)
	}
	ix, err := Open(path)
	if err != nil {
		t.Fatalf("открытие %s: %v", path, err)
	}
	if ix.Places() < 100000 {
		t.Fatalf("в индексе %d мест — похоже, собран из неполного дампа", ix.Places())
	}
	cases := []struct {
		q, tz, country string
	}{
		{"Москва", "Europe/Moscow", "RU"},
		{"Санкт-Петербург", "Europe/Moscow", "RU"},
		{"Новосибирск", "Asia/Novosibirsk", "RU"},
		{"Владивосток", "Asia/Vladivostok", "RU"},
		{"Лондон", "Europe/London", "GB"},
		{"London", "Europe/London", "GB"},
		{"Нью-Йорк", "America/New_York", "US"},
	}
	for _, c := range cases {
		res, ok := ix.Lookup(c.q)
		if !ok {
			t.Errorf("Lookup(%q): не найдено в реальном дампе", c.q)
			continue
		}
		if res.Place.Timezone != c.tz {
			t.Errorf("Lookup(%q) → %s (tz %s), ожидался %s", c.q, res.Place.Name, res.Place.Timezone, c.tz)
		}
		if c.country != "" && res.Place.Country != c.country {
			t.Errorf("Lookup(%q) → страна %s, ожидалась %s", c.q, res.Place.Country, c.country)
		}
	}
	// Мусор не должен находиться: раньше Nominatim отдавал для «Ззззъъъ» Японию.
	for _, junk := range []string{"Ззззъъъ", "qwertyuiopasdf"} {
		if res, ok := ix.Lookup(junk); ok {
			t.Errorf("Lookup(%q) нашёл %s (%s) — так быть не должно", junk, res.Place.Name, res.Place.Country)
		}
	}
}
