package interactive

import "testing"

func TestNameMatchesCity(t *testing.T) {
	ok := []struct{ query, display string }{
		{"Москва", "Москва, Россия"},
		{"москва", "Москва, городской округ Москва, Россия"},
		{"Королёв", "Королев, Московская область, Россия"},
		{"Санкт-Петербург", "Санкт-Петербург, Россия"},
		{"New York", "New York, United States"},
		{"São Paulo", "São Paulo, Brasil"},
	}
	for _, c := range ok {
		if !nameMatchesCity(c.query, c.display) {
			t.Errorf("nameMatchesCity(%q, %q) = false, ожидалось true", c.query, c.display)
		}
	}
	bad := []struct{ query, display string }{
		{"Ззззъъъ", "Минато, Токио, Япония"},
		{"Москва", "Московская область, Россия"},
		{"qwertyuiopasdf", "Springfield, Illinois, United States"},
		{"", "Москва, Россия"},
	}
	for _, c := range bad {
		if nameMatchesCity(c.query, c.display) {
			t.Errorf("nameMatchesCity(%q, %q) = true, ожидалось false", c.query, c.display)
		}
	}
}

func TestTZOffsetAt(t *testing.T) {
	cases := []struct {
		tz, date string
		want     float64
	}{
		{"Asia/Novosibirsk", "1990-05-15", 8}, // летнее время СССР
		{"Asia/Novosibirsk", "1990-01-15", 7},
		{"Europe/London", "1990-07-15", 1},
		{"Europe/London", "2026-01-15", 0},
		{"Europe/Moscow", "2014-01-15", 4}, // до перехода на +3
		{"Europe/Moscow", "2026-01-15", 3},
	}
	for _, c := range cases {
		if got := tzOffsetAt(c.tz, c.date, "12:00"); got != c.want {
			t.Errorf("tzOffsetAt(%s, %s)=%v, ожидалось %v", c.tz, c.date, got, c.want)
		}
	}
	// Неизвестная зона не должна превращаться в «московское время».
	if got := tzOffsetAt("No/Such_Zone", "2026-01-15", "12:00"); got != 0 {
		t.Errorf("неизвестная зона: offset=%v, ожидался 0", got)
	}
}
