// pkg/output/console.go
package output

import (
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/damirmur/swisseph_build/pkg/astro"
	"github.com/damirmur/swisseph_build/pkg/interpreter"
)

type ConsoleRenderer struct{}

func (r *ConsoleRenderer) Render(ctx context.Context, result *astro.AstroResult, w io.Writer) error {
	if result == nil {
		return fmt.Errorf("данные отсутствуют")
	}

	switch result.Type {
	case "natal":
		r.renderNatal(result, w)
	case "synastry":
		r.renderSynastry(result, w)
	case "period":
		r.renderPeriod(result, w)
	case "calendar":
		r.renderCalendar(result, w)
	}

	return nil
}

// 1. Вывод Натальной карты
func (r *ConsoleRenderer) renderNatal(res *astro.AstroResult, w io.Writer) {
	fmt.Fprintln(w, "\n====================================================")
	fmt.Fprintln(w, "   НАТАЛЬНАЯ КАРТА")
	fmt.Fprintln(w, "====================================================")
	fmt.Fprintf(w, "Дата и время (UTC): %s\n", res.Timestamp.Format("2006-01-02 15:04"))
	fmt.Fprintln(w, "----------------------------------------------------")

	// Таблица планет
	fmt.Fprintln(w, "\n[ ПОЛОЖЕНИЕ ПЛАНЕТ ]")
	fmt.Fprintln(w, "| Планета    | Долгота      | Скорость   |")
	for _, p := range res.Planets {
		fmt.Fprintf(w, "| %-10s | %-12s | %-10s |\n", p.Name, astro.FormatDegree(p.Longitude), astro.FormatSpeed(p.Speed))
	}

	// Таблица домов
	fmt.Fprintln(w, "\n[ ПОЛОЖЕНИЕ ДОМОВ ]")
	for _, cusp := range res.Houses {
		fmt.Fprintf(w, "Дом %-2d: %s\n", cusp.Number, astro.FormatDegree(cusp.Longitude))
	}

	// Планеты в домах
	fmt.Fprintln(w, "\n[ ПЛАНЕТЫ В ДОМАХ ]")

	// Сортировка по дому, затем по ID планеты
	sortedPlanets := make([]astro.Position, len(res.Planets))
	copy(sortedPlanets, res.Planets)
	sort.Slice(sortedPlanets, func(i, j int) bool {
		if sortedPlanets[i].House == sortedPlanets[j].House {
			return sortedPlanets[i].ID < sortedPlanets[j].ID
		}
		return sortedPlanets[i].House < sortedPlanets[j].House
	})

	for _, p := range sortedPlanets {
		fmt.Fprintf(w, "%-10s -> Дом %d\n", p.Name, p.House)
	}

	// Аспекты (уже отсортированы в calculator.go)
	if len(res.Aspects) > 0 {
		fmt.Fprintln(w, "\n[ АСПЕКТЫ ]")
		for _, a := range res.Aspects {
			fmt.Fprintf(w, "%-10s %-12s %-10s (орб: %.2f)\n", a.Planet1, a.Type, a.Planet2, a.Orb)
		}
	}

	// Интерпретация
	fmt.Fprintln(w, "\n[ ИНТЕРПРЕТАЦИЯ ]")
	prompt := interpreter.GenerateInterpretation(res, "не указан", 0)
	fmt.Fprintln(w, "Сформирован запрос для интерпретации:")
	fmt.Fprintln(w, prompt)
}

// 2. Вывод Синастрии / Транзитов
func (r *ConsoleRenderer) renderSynastry(res *astro.AstroResult, w io.Writer) {
	fmt.Fprintln(w, "\n====================================================")
	fmt.Fprintf(w, "   СИНАСТРИЯ / ТРАНЗИТ | Дата события: %s\n", res.Timestamp.Format("2006-01-02 15:04"))
	fmt.Fprintln(w, "====================================================")

	if len(res.Aspects) > 0 {
		fmt.Fprintln(w, "\n[ МЕЖКАРТОВЫЕ АСПЕКТЫ ]")
		fmt.Fprintln(w, "------------------------------------------------------------------")
		fmt.Fprintln(w, "| Карта Б (Транзит)   | Аспект       | Карта А (Натал)     | Орбис   |")
		fmt.Fprintln(w, "------------------------------------------------------------------")
		for _, a := range res.Aspects {
			fmt.Fprintf(w, "| %-19s | %-12s | %-19s | %-7.4f |\n",
				a.Planet1, a.Type, a.Planet2, a.Orb)
		}
		fmt.Fprintln(w, "------------------------------------------------------------------")
	} else {
		fmt.Fprintln(w, "\nМежкартовых аспектов не обнаружено.")
	}
}

// 3. Вывод Положений за период
func (r *ConsoleRenderer) renderPeriod(res *astro.AstroResult, w io.Writer) {
	fmt.Fprintln(w, "\n====================================================")
	fmt.Fprintln(w, "   ДИНАМИКА ПЛАНЕТ ЗА ПЕРИОД")
	fmt.Fprintln(w, "====================================================")

	for _, slice := range res.Slices {
		fmt.Fprintf(w, "\n> Временной срез: %s\n", slice.Timestamp.Format("2006-01-02 15:04"))
		r.printPlanetsShort(slice.Planets, w)
	}
}

// Вспомогательный метод структуры ConsoleRenderer
func (r *ConsoleRenderer) printPlanetsShort(planets []astro.Position, w io.Writer) {
	for _, p := range planets {
		fmt.Fprintf(w, "  %-10s: %s |", p.Name, astro.FormatDegree(p.Longitude))
	}
	fmt.Fprintln(w)
}

// 4. Вывод Календаря астрособытий
func (r *ConsoleRenderer) renderCalendar(res *astro.AstroResult, w io.Writer) {
	fmt.Fprintln(w, "\n====================================================")
	if res.Timezone != "" {
		fmt.Fprintf(w, "   КАЛЕНДАРЬ АСТРОЛОГИЧЕСКИХ СОБЫТИЙ (%s)\n", res.Timezone)
	} else {
		fmt.Fprintln(w, "   КАЛЕНДАРЬ АСТРОЛОГИЧЕСКИХ СОБЫТИЙ")
	}
	fmt.Fprintln(w, "====================================================")

	if len(res.Events) == 0 {
		fmt.Fprintln(w, "Событий за указанный период не найдено.")
		return
	}

	for i, event := range res.Events {
		// Вызываем общую функцию форматирования напрямую
		fmt.Fprintf(w, "%03d. %s\n", i+1, FormatCalendarEventText(event))
	}

	fmt.Fprintln(w, "====================================================")
}
