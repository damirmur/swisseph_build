package astro

import (
	"reflect"
	"testing"
)

func TestComputeBalance(t *testing.T) {
	planets := []PlanetPoint{
		{ID: 0, Longitude: 0},    // Aries: fire / cardinal
		{ID: 1, Longitude: 30},   // Taurus: earth / fixed
		{ID: 2, Longitude: 60},   // Gemini: air / mutable
		{ID: 3, Longitude: 90},   // Cancer: water / cardinal
		{ID: 9, Longitude: 270},  // Capricorn: earth / cardinal
		{ID: 10, Longitude: 120}, // North Node — не учитываем
		{ID: 12, Longitude: 150}, // Lilith — не учитываем
	}
	b := ComputeBalance(planets)
	if want := []int{0, 1, 2, 3, 9}; !reflect.DeepEqual(b.Planets, want) {
		t.Fatalf("planets = %v, ожидалось %v", b.Planets, want)
	}
	wantEl := map[string]int{"fire": 1, "earth": 2, "air": 1, "water": 1}
	if !reflect.DeepEqual(b.Elements, wantEl) {
		t.Fatalf("elements = %v, ожидалось %v", b.Elements, wantEl)
	}
	wantCr := map[string]int{"cardinal": 3, "fixed": 1, "mutable": 1}
	if !reflect.DeepEqual(b.Crosses, wantCr) {
		t.Fatalf("crosses = %v, ожидалось %v", b.Crosses, wantCr)
	}
	if got := b.Grid["fire"]["cardinal"]; !reflect.DeepEqual(got, []int{0}) {
		t.Errorf("fire/cardinal = %v", got)
	}
	if got := b.Grid["earth"]["fixed"]; !reflect.DeepEqual(got, []int{1}) {
		t.Errorf("earth/fixed = %v", got)
	}
	if got := b.Grid["air"]["mutable"]; !reflect.DeepEqual(got, []int{2}) {
		t.Errorf("air/mutable = %v", got)
	}
	if got := b.Grid["water"]["cardinal"]; !reflect.DeepEqual(got, []int{3}) {
		t.Errorf("water/cardinal = %v", got)
	}
	if got := b.Grid["earth"]["cardinal"]; !reflect.DeepEqual(got, []int{9}) {
		t.Errorf("earth/cardinal = %v", got)
	}
}

func TestComputeBalanceEmpty(t *testing.T) {
	b := ComputeBalance(nil)
	if len(b.Planets) != 0 {
		t.Fatalf("planets = %v", b.Planets)
	}
	for _, el := range []string{"fire", "earth", "air", "water"} {
		for _, cr := range []string{"cardinal", "fixed", "mutable"} {
			if got := b.Grid[el][cr]; got == nil || len(got) != 0 {
				t.Fatalf("пустая ячейка %s/%s = %v, ожидался пустой массив", el, cr, got)
			}
		}
	}
}

func TestToNatalDataIncludesBalance(t *testing.T) {
	res := &AstroResult{Planets: []Position{
		{ID: 0, Longitude: 10}, {ID: 1, Longitude: 40}, {ID: 10, Longitude: 70},
	}}
	d := ToNatalData(res)
	if len(d.Balance.Planets) != 2 {
		t.Fatalf("в баланс попали не те планеты: %v", d.Balance.Planets)
	}
	if d.Balance.Elements["fire"] != 1 || d.Balance.Elements["earth"] != 1 {
		t.Fatalf("elements = %v", d.Balance.Elements)
	}
}

func TestToSynastryDataBalanceBothCharts(t *testing.T) {
	res := &AstroResult{
		Planets:  []Position{{ID: 0, Longitude: 0}},
		PlanetsB: []Position{{ID: 1, Longitude: 30}},
	}
	d := ToSynastryData(res)
	if len(d.Chart1.Balance.Planets) != 1 || d.Chart1.Balance.Planets[0] != 0 {
		t.Fatalf("chart1 balance = %+v", d.Chart1.Balance)
	}
	if len(d.Chart2.Balance.Planets) != 1 || d.Chart2.Balance.Planets[0] != 1 {
		t.Fatalf("chart2 balance = %+v", d.Chart2.Balance)
	}
}
