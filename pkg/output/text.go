// pkg/output/text.go
package output

import (
	"context"
	"fmt"
	"io"

	"github.com/damirmur/swisseph_build/pkg/astro"
)

// TextRenderer отвечает за генерацию файлов .txt с текстовыми отчетами
type TextRenderer struct{}

func (r *TextRenderer) Render(ctx context.Context, result *astro.AstroResult, w io.Writer) error {
	if result == nil {
		return fmt.Errorf("данные для генерации текста отсутствуют")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	switch result.Type {
	case "natal":
		r.writeNatalText(result, w)
	case "synastry":
		r.writeSynastryText(result, w)
	case "period":
		r.writePeriodText(result, w)
	case "calendar":
		r.writeCalendarText(result, w)
	}

	return nil
}

func (r *TextRenderer) writeNatalText(res *astro.AstroResult, w io.Writer) {
	fmt.Fprintln(w, "==================================================================")
	fmt.Fprintf(w, "АНАЛИТИЧЕСКИЙ ОТЧЕТ: НАТАЛЬНАЯ КАРТА\nДата расчета (UTC): %s\n", res.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Fprintln(w, "==================================================================")

	fmt.Fprintln(w, "\n[ПОЛОЖЕНИЕ НЕБЕСНЫХ ТЕЛ В ЗНАКАХ И ДОМАХ]")
	for _, p := range res.Planets {
		signIdx := int(p.Longitude / 30.0)
		signDeg := p.Longitude - float64(signIdx*30)
		signName := astro.GetSignRuGenitive(signIdx)

		fmt.Fprintf(w, "• %-10s находится в %02d° %-8s (Долгота: %7.3f°, Дом: %d)\n",
			p.Name, int(signDeg), signName, p.Longitude, p.House)
	}

	if len(res.Aspects) > 0 {
		fmt.Fprintln(w, "\n[ГЕОМЕТРИЧЕСКИЕ СВЯЗИ И АСПЕКТЫ]")
		for _, a := range res.Aspects {
			fmt.Fprintf(w, "  - Взаимодействие %s и %s через аспект %s (Точность/Орбис: %.2f°)\n",
				a.Planet1, a.Planet2, astro.GetAspectRu(a.Type), a.Orb)
		}
	}
}

func (r *TextRenderer) writeSynastryText(res *astro.AstroResult, w io.Writer) {
	fmt.Fprintln(w, "==================================================================")
	fmt.Fprintf(w, "ОТЧЕТ СОВМЕСТИМОСТИ (СИНАСТРИЯ / ТРАНЗИТ)\nДата анализа: %s\n", res.Timestamp.Format("2006-01-02 15:04"))
	fmt.Fprintln(w, "==================================================================")

	if len(res.Aspects) > 0 {
		fmt.Fprintln(w, "\n[МЕЖКАРТОВЫЕ ВЛИЯНИЯ И АСПЕКТЫ]")
		for _, a := range res.Aspects {
			fmt.Fprintf(w, "  • Динамическая точка %s формирует аспект %s к натальной планете %s (Орбис: %.2f°)\n",
				a.Planet1, astro.GetAspectRu(a.Type), a.Planet2, a.Orb)
		}
	} else {
		fmt.Fprintln(w, "\nТочных межкартовых аспектных взаимодействий на данный период не обнаружено.")
	}
}

func (r *TextRenderer) writePeriodText(res *astro.AstroResult, w io.Writer) {
	fmt.Fprintln(w, "==================================================================")
	fmt.Fprintln(w, "ОТЧЕТ: ХРОНОЛОГИЧЕСКОЕ ДВИЖЕНИЕ ПЛАНЕТ")
	fmt.Fprintln(w, "==================================================================")

	for _, slice := range res.Slices {
		fmt.Fprintf(w, "\n[Период: %s]\n", slice.Timestamp.Format("2006-01-02 15:04"))
		for _, p := range slice.Planets {
			fmt.Fprintf(w, "  %-10s -> %7.3f° |", p.Name, p.Longitude)
		}
		fmt.Fprintln(w)
	}
}

func (r *TextRenderer) writeCalendarText(res *astro.AstroResult, w io.Writer) {
	fmt.Fprintln(w, "==================================================================")
	if res.Timezone != "" {
		fmt.Fprintf(w, "АСТРОЛОГИЧЕСКИЙ КАЛЕНДАРЬ СОБЫТИЙ (%s)\n", res.Timezone)
	} else {
		fmt.Fprintln(w, "АСТРОЛОГИЧЕСКИЙ КАЛЕНДАРЬ СОБЫТИЙ")
	}
	fmt.Fprintln(w, "==================================================================")

	for i, event := range res.Events {
		// Вызываем общую функцию форматирования напрямую
		fmt.Fprintf(w, "%03d. %s\n", i+1, FormatCalendarEventText(event))
	}
}
