package tests

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/damirmur/swisseph_build/pkg/astro"
)

func celestialShortestDiff(a, b float64) float64 {
	d := math.Abs(a - b)
	if d > 180 {
		d = 360 - d
	}
	return d
}

// TestNatalPeriodExactAspectPrecision проверяет, что время события exact
// уточнено по эфемеридам, а не привязано к часовому срезу периода:
// пересчёт орба в этот момент должен давать практически ноль.
func TestNatalPeriodExactAspectPrecision(t *testing.T) {
	ephePath, err := filepath.Abs("../ephe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ephePath); err != nil {
		t.Skipf("каталог эфемерид %q не найден", ephePath)
	}
	calc := astro.NewCalculator(ephePath)
	defer calc.Close()
	ctx := context.Background()
	natalTime := time.Date(1990, 1, 1, 9, 0, 0, 0, time.UTC)
	start := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)
	res, err := calc.ComputeNatalPeriod(ctx, natalTime, 55.75, 37.62, "P", start, end, 1,
		[]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12}, 1.0, []float64{0, 60, 90, 120, 180}, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	natalLon := map[int]float64{}
	for _, p := range res.Natal.Planets {
		natalLon[p.ID] = p.Longitude
	}
	exactCount := 0
	for _, e := range res.Aspects {
		if e.Kind != "exact" {
			continue
		}
		exactCount++
		evTime, err := time.Parse(time.RFC3339, e.Date)
		if err != nil {
			t.Fatal(err)
		}
		chart, err := calc.ComputeNatal(ctx, evTime, 55.75, 37.62, "P")
		if err != nil {
			t.Fatal(err)
		}
		var transitLon float64
		for _, p := range chart.Planets {
			if p.ID == e.TransitID {
				transitLon = p.Longitude
				break
			}
		}
		orb := math.Abs(celestialShortestDiff(transitLon, natalLon[e.NatalID]) - e.Angle)
		t.Logf("%s transit=%d natal=%d angle=%.0f orb=%.6f", e.Date, e.TransitID, e.NatalID, e.Angle, orb)
		if evTime.Before(start) || evTime.After(end) {
			t.Errorf("точное событие %s вышло за период", e.Date)
		}
		if orb > 0.002 {
			t.Errorf("орб в точном событии %s слишком велик: %.6f", e.Date, orb)
		}
	}
	if exactCount < 3 {
		t.Fatalf("точных событий слишком мало: %d", exactCount)
	}
}
