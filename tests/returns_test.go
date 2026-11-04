package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/damirmur/swisseph_build/pkg/astro"
)

// returnsHouseLat/Lon — координаты натала в тестах возвратов.
const (
	returnsLat = 55.75
	returnsLon = 37.61
)

// planetLon находит долготу планеты с заданным id в натальной карте.
func planetLon(t *testing.T, calc *astro.Calculator, at time.Time, id int) float64 {
	t.Helper()
	res, err := calc.ComputeNatal(context.Background(), at, returnsLat, returnsLon, "P")
	if err != nil {
		t.Fatalf("натальная карта на %s: %v", at.Format(time.RFC3339), err)
	}
	for _, p := range res.Planets {
		if p.ID == id {
			return p.Longitude
		}
	}
	t.Fatalf("в карте на %s нет планеты id=%d", at.Format(time.RFC3339), id)
	return 0
}

// TestReturnsMoments — соляры и лунары: порядок моментов, периоды обращения
// и точность попадания на натальную долготу (а не просто «рядом»).
func TestReturnsMoments(t *testing.T) {
	ephePath, err := filepath.Abs("../ephe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ephePath); err != nil {
		t.Skipf("каталог эфемерид %q не найден", ephePath)
	}
	calc := astro.NewCalculator(ephePath)
	defer calc.Close()

	natal := time.Date(1990, 6, 15, 9, 0, 0, 0, time.UTC)
	ref := time.Date(2026, 9, 27, 9, 0, 0, 0, time.UTC)

	res, err := calc.ComputeReturns(natal, ref)
	if err != nil {
		t.Fatalf("ComputeReturns: %v", err)
	}

	// Натальные долготы в результате — те же, что даёт обычный натальный расчёт.
	if got, want := res.NatalSunLon, planetLon(t, calc, natal, 0); celestialShortestDiff(got, want) > 1e-6 {
		t.Errorf("натальная долгота Солнца: %v, ожидалась %v", got, want)
	}
	if got, want := res.NatalMoonLon, planetLon(t, calc, natal, 1); celestialShortestDiff(got, want) > 1e-6 {
		t.Errorf("натальная долгота Луны: %v, ожидалась %v", got, want)
	}

	for _, c := range []struct {
		name       string
		pair       astro.ReturnPair
		id         int
		natalLon   float64
		minDays    float64
		maxDays    float64
		lonEpsilon float64
	}{
		{"соляр", res.Solar, 0, res.NatalSunLon, 364, 367, 1e-4},
		// Луна проходит ~0.00015°/с, а dateToJulian округляет момент до целой
		// секунды — на пересчёте это до ~1.5e-4°, поэтому допуск шире.
		{"лунар", res.Lunar, 1, res.NatalMoonLon, 27.0, 27.7, 5e-4},
	} {
		if !c.pair.Prev.Before(ref) {
			t.Errorf("%s: предыдущий возврат %s не раньше ref %s", c.name, c.pair.Prev, ref)
		}
		if !c.pair.Next.After(ref) {
			t.Errorf("%s: следующий возврат %s не позже ref %s", c.name, c.pair.Next, ref)
		}
		if days := c.pair.Next.Sub(c.pair.Prev).Hours() / 24; days < c.minDays || days > c.maxDays {
			t.Errorf("%s: между возвратами %.2f дн., ожидалось %.1f–%.1f", c.name, days, c.minDays, c.maxDays)
		}
		// В момент возврата планета должна стоять ровно на натальной долготе.
		if lon := planetLon(t, calc, c.pair.Next, c.id); celestialShortestDiff(lon, c.natalLon) > c.lonEpsilon {
			t.Errorf("%s: в момент возврата долгота %v, натальная %v (разница %v°)",
				c.name, lon, c.natalLon, celestialShortestDiff(lon, c.natalLon))
		}
		if lon := planetLon(t, calc, c.pair.Prev, c.id); celestialShortestDiff(lon, c.natalLon) > c.lonEpsilon {
			t.Errorf("%s: в предыдущий возврат долгота %v, натальная %v", c.name, lon, c.natalLon)
		}
	}

	// ref ровно в момент возврата: он остаётся «следующим», а «предыдущий» —
	// прошлогодний/прошломесячный, а не тот же самый момент.
	again, err := calc.ComputeReturns(natal, res.Solar.Next)
	if err != nil {
		t.Fatalf("ComputeReturns(ref=возврат): %v", err)
	}
	if d := again.Solar.Next.Sub(res.Solar.Next); d > 2*time.Second || d < -2*time.Second {
		t.Errorf("при ref=возврат следующий соляр сдвинулся на %v", d)
	}
	if !again.Solar.Prev.Before(res.Solar.Next) {
		t.Errorf("при ref=возврат предыдущий соляр %s не раньше %s", again.Solar.Prev, res.Solar.Next)
	}
	if d := again.Solar.Prev.Sub(res.Solar.Prev); d > 2*time.Minute || d < -2*time.Minute {
		t.Errorf("при ref=возврат предыдущий соляр %s, ожидался около %s", again.Solar.Prev, res.Solar.Prev)
	}
}
