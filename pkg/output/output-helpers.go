// pkg/output/output-helpers.go
package output

import (
	"fmt"
	"time"

	"github.com/damirmur/swisseph_build/pkg/astro"
)

// compactDayTime сводит момент tISO к "ЧЧ:ММ", если он в том же дне, что startISO,
// иначе к "ЧЧ:ММ ДД.ММ". При ошибке парсинга возвращает tISO как есть.
func compactDayTime(startISO, tISO string) string {
	tStart, errStart := time.Parse("2006-01-02T15:04", startISO)
	tEnd, errEnd := time.Parse("2006-01-02T15:04", tISO)
	if errEnd != nil {
		return tISO
	}
	if errStart == nil &&
		tStart.Year() == tEnd.Year() && tStart.Month() == tEnd.Month() && tStart.Day() == tEnd.Day() {
		return tEnd.Format("15:04")
	}
	return tEnd.Format("15:04 02.01")
}

// FormatCalendarEventText превращает сырое астрологическое событие в красивую строку для вывода в консоль или файлы.
func FormatCalendarEventText(e astro.CalendarEvent) string {
	timeStr := e.Date
	if t, err := time.Parse("2006-01-02T15:04", e.Date); err == nil {
		timeStr = t.Format("02.01.2006 15:04")
	}

	getSignVal := func() int {
		if e.Sign != nil {
			return *e.Sign
		}
		return 0
	}

	switch e.Type {
	case "noC": // Луна без курса (Холостая): старт — последний аспект,
		// конец (VoidEnd) — первый аспект в новом знаке, ChangeSign — смена знака между ними
		endStr := e.VoidEnd
		if endStr == "" {
			endStr = e.ChangeSign // фолбэк на старое поведение
		}
		untilStr := compactDayTime(e.Date, endStr)
		if e.VoidEnd != "" && e.ChangeSign != "" {
			return fmt.Sprintf("[%s] Луна без курса (холостая) до %s (смена знака %s)",
				timeStr, untilStr, compactDayTime(e.Date, e.ChangeSign))
		}

		return fmt.Sprintf("[%s] Луна без курса (холостая) до %s", timeStr, untilStr)

	case "chS": // Ингрессия
		planet := "Планета"
		if len(e.Planets) > 0 {
			planet = astro.GetPlanetRu(e.Planets[0])
		}
		return fmt.Sprintf("[%s] Ингрессия: %s переходит в знак %s", timeStr, planet, astro.GetSignRuGenitive(getSignVal()))

	case "r": // Разворот движения
		planet := "Планета"
		if len(e.Planets) > 0 {
			planet = astro.GetPlanetRu(e.Planets[0])
		}
		motionType := "разворачивается в директное движение"
		if e.Aspect == "R" {
			motionType = "разворачивается в ретроградное движение"
		}
		return fmt.Sprintf("[%s] Разворот: %s %s", timeStr, planet, motionType)

	case "lunD": // Лунные дни
		return fmt.Sprintf("[%s] %d-й лунный день", timeStr, getSignVal())

	case "exA": // Аспекты
		p1, p2 := "??", "??"
		if len(e.Planets) > 1 {
			p1 = astro.GetPlanetRu(e.Planets[0])
			p2 = astro.GetPlanetRu(e.Planets[1])
		}

		return fmt.Sprintf("[%s] Аспект: %s - %s (%s)", timeStr, p1, p2, astro.GetAspectRu(e.Aspect))

	default:
		return fmt.Sprintf("[%s] Событие %s %v", timeStr, e.Type, e.Planets)
	}
}
