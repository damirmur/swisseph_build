package interpreter

import (
	"fmt"
	"sort"
	"github.com/damirmur/swisseph_build/pkg/astro"
)

func GenerateInterpretation(res *astro.AstroResult, gender string, age int) string {
	prompt := fmt.Sprintf("Проинтерпретируй натальную карту (Пол: %s, Возраст: %d):\n\n", gender, age)
	
	prompt += "Данные:\n"
	
	// Сортируем копию планет по домам, затем по ID
	sortedPlanets := make([]astro.Position, len(res.Planets))
	copy(sortedPlanets, res.Planets)
	sort.Slice(sortedPlanets, func(i, j int) bool {
		if sortedPlanets[i].House == sortedPlanets[j].House {
			return sortedPlanets[i].ID < sortedPlanets[j].ID
		}
		return sortedPlanets[i].House < sortedPlanets[j].House
	})

	for _, p := range sortedPlanets {
		prompt += fmt.Sprintf("- %s в %s (Дом %d)\n", p.Name, astro.FormatDegree(p.Longitude), p.House)
	}
	
	prompt += "\nАспекты:\n"
	for _, a := range res.Aspects {
		prompt += fmt.Sprintf("- %s %s %s (орб: %.2f)\n", a.Planet1, a.Type, a.Planet2, a.Orb)
	}

	prompt += "\nЗадача: Проинтерпретируй каждую планету, каждый аспект и положение планет в домах. В конце составь общий психологический портрет."
	return prompt
}
