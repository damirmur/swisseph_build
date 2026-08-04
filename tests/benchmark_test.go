package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/damirmur/swisseph_build/pkg/astro"
)

func BenchmarkCalendarYear(b *testing.B) {
	ephePath, err := filepath.Abs("../ephe")
	if err != nil {
		b.Fatal(err)
	}
	if _, err := os.Stat(ephePath); err != nil {
		b.Skipf("каталог эфемерид %q не найден", ephePath)
	}

	tStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Add(-24 * time.Hour)
	tEnd := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC).Add(24 * time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc := astro.NewCalculator(ephePath)
		_, err := calc.ComputeCalendar(context.Background(), tStart, tEnd, time.UTC)
		calc.Close()
		if err != nil {
			b.Fatal(err)
		}
	}
}
