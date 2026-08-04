package astro

import "math"

// AspectConfig задает параметры для поиска аспекта
type AspectConfig struct {
	Name  string
	Angle float64
	Orb   float64
}

var MajorAspects = []AspectConfig{
	{Name: "Conjunction", Angle: 0, Orb: 6.0},
	{Name: "Sextile", Angle: 60, Orb: 4.0},
	{Name: "Square", Angle: 90, Orb: 5.0},
	{Name: "Trine", Angle: 120, Orb: 6.0},
	{Name: "Opposition", Angle: 180, Orb: 4.0},
}

// CalculateAspect проверяет наличие аспекта между двумя долготами (0-360)
func CalculateAspect(lon1, lon2 float64) (string, float64, float64, bool) {
	diff := getShortestDiff(lon1, lon2)

	for _, asp := range MajorAspects {
		exactDiff := math.Abs(diff - asp.Angle)
		if exactDiff <= asp.Orb {
			return asp.Name, asp.Angle, exactDiff, true
		}
	}
	return "", 0, 0, false
}

// aspectAngle возвращает угол аспекта по его имени (Conjunction, Sextile, ...) из MajorAspects.
func aspectAngle(name string) float64 {
	for _, a := range MajorAspects {
		if a.Name == name {
			return a.Angle
		}
	}
	return 0
}
