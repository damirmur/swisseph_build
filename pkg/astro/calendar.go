package astro

/*
#cgo CFLAGS: -I${SRCDIR}/../../../swisseph_build/include
#cgo LDFLAGS: -L${SRCDIR}/../../../swisseph_build/lib -lswe -lm
#include "swephexp.h"
*/
import "C"

import (
	"context"
	"math"
	"slices"
	"strconv"
	"time"
	"unsafe"
)

// getShortestDiff возвращает кратчайшее угловое расстояние между двумя долготами (0-360)
func getShortestDiff(lon1, lon2 float64) float64 {
	diff := math.Abs(lon1 - lon2)
	if diff > 180 {
		diff = 360 - diff
	}
	return diff
}

// normalizeAngle приводит угол к диапазону (-180, 180]
func normalizeAngle(a float64) float64 {
	a = math.Mod(a+180, 360)
	if a < 0 {
		a += 360
	}
	return a - 180
}

// getSignedDiff возвращает направленное расстояние (по ходу зодиака) от lon2 до lon1 в диапазоне (-180, 180]
func getSignedDiff(lon1, lon2 float64) float64 {
	return normalizeAngle(lon1 - lon2)
}

// aspectCrossed определяет, пересекло ли угловое расстояние планет целевой аспектный угол между двумя соседними отсчетами.
// Для 60°/90°/120° достаточно классической проверки пересечения кратчайшего расстояния.
// Соединение (0°) и оппозиция (180°) — экстремумы кратчайшего расстояния (локальный минимум/максимум),
// они не «пересекаются», а только касаются: их ловим по смене знака направленной разности долгот,
// различая случаи по величине расстояния (при 0° планеты рядом, при 180° — напротив).
func aspectCrossed(prevLon1, prevLon2, currLon1, currLon2, angle float64) bool {
	prevShortest := getShortestDiff(prevLon1, prevLon2)
	currShortest := getShortestDiff(currLon1, currLon2)

	if angle > 0 && angle < 180 {
		return (prevShortest <= angle && currShortest >= angle) ||
			(prevShortest >= angle && currShortest <= angle)
	}

	if prevShortest == angle || currShortest == angle {
		return true
	}

	prevSigned := getSignedDiff(prevLon1, prevLon2)
	currSigned := getSignedDiff(currLon1, currLon2)
	if prevSigned == 0 || currSigned == 0 {
		return true
	}
	if prevSigned*currSigned < 0 {
		if angle == 0 {
			return prevShortest <= 90 && currShortest <= 90
		}
		return prevShortest >= 90 && currShortest >= 90
	}
	return false
}

// ComputeCalendar выполняет генерацию календаря событий без дублирования
func (c *Calculator) ComputeCalendar(ctx context.Context, start, end time.Time, loc *time.Location) (*AstroResult, error) {
	if loc == nil {
		loc = time.UTC
	}

	planetIDs := GetPlanetIDs()

	var events []CalendarEvent
	step := 24 * time.Hour

	type PlanetState struct {
		Longitude float64
		SignIndex int
		Speed     float64 // Добавляем скорость по долготе (tmpXX[3])
	}

	prevState := make(map[int]PlanetState)

	tjdUt := dateToJulian(start)
	var tmpXX [6]C.double
	var tmpSerr [256]C.char

	// Инициализируем начальное состояние планет
	for _, id := range planetIDs {
		if C.swe_calc_ut(tjdUt, C.int(id), C.SEFLG_SWIEPH|C.SEFLG_SPEED, (*C.double)(unsafe.Pointer(&tmpXX)), (*C.char)(unsafe.Pointer(&tmpSerr))) >= 0 {
			lon := float64(tmpXX[0])
			prevState[id] = PlanetState{
				Longitude: lon,
				SignIndex: int(lon / 30.0),
				Speed:     float64(tmpXX[3]), // Скорость в градусах/день
			}
		}
	}

	// 1. Основной цикл: ингрессии (смена знаков) и аспекты планет
	for currentTime := start.Add(step); !currentTime.After(end); currentTime = currentTime.Add(step) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		tjdUt = dateToJulian(currentTime)
		currState := make(map[int]PlanetState)

		// Сначала рассчитываем координаты всех планет на текущий шаг
		for _, id := range planetIDs {
			if C.swe_calc_ut(tjdUt, C.int(id), C.SEFLG_SWIEPH|C.SEFLG_SPEED, (*C.double)(unsafe.Pointer(&tmpXX)), (*C.char)(unsafe.Pointer(&tmpSerr))) >= 0 {
				lon := float64(tmpXX[0])
				currState[id] = PlanetState{
					Longitude: lon,
					SignIndex: int(lon / 30.0),
					Speed:     float64(tmpXX[3]), // Скорость в градусах/день
				}
			}
		}

		// Проверяем ингрессии (смену знаков) и смену направления
		for _, id := range planetIDs {
			prev := prevState[id]
			curr := currState[id]

			// Проверяем ингрессии
			if curr.SignIndex != prev.SignIndex {
				exactTime, _ := findExactSignChange(id, curr.SignIndex, currentTime.Add(-step), currentTime)
				signVal := curr.SignIndex
				events = append(events, CalendarEvent{
					Date:    exactTime.In(loc).Format("2006-01-02T15:04"),
					Type:    "chS",
					Planets: []string{strconv.Itoa(id)},
					Sign:    &signVal,
					Degrees: []int{curr.SignIndex * 30},
				})
			}

			// Проверяем смену направления движения (исключаем Солнце ID=0 и Луну ID=1)
			if id != 0 && id != 1 {
				// Если скорость пересекла ноль (изменила знак)
				if (prev.Speed > 0 && curr.Speed < 0) || (prev.Speed < 0 && curr.Speed > 0) {
					exactStationTime, _ := findExactStationTime(id, currentTime.Add(-step), currentTime)

					// Определяем характер движения: R (Ретро) или D (Директ)
					direction := "D"
					if curr.Speed < 0 {
						direction = "R"
					}
					signVal := curr.SignIndex
					events = append(events, CalendarEvent{
						Date:    exactStationTime.In(loc).Format("2006-01-02T15:04"),
						Type:    "r", // Change Direction
						Planets: []string{strconv.Itoa(id)},
						Sign:    &signVal,
						Aspect:  direction, // Используем поле Aspect для хранения "R" или "D" (либо настройте под вашу структуру)
						Degrees: []int{int(curr.Longitude)},
					})
				}
			}
		}

		// Проверяем мажорные аспекты планет по пересечению точного угла
		for i := 0; i < len(planetIDs); i++ {
			for j := i + 1; j < len(planetIDs); j++ {
				id1 := planetIDs[i]
				id2 := planetIDs[j]

				// Исключаем Луну (ID=1) из суточного поиска аспектных событий
				if id1 == 1 || id2 == 1 {
					continue
				}

				prevLon1 := prevState[id1].Longitude
				prevLon2 := prevState[id2].Longitude
				currLon1 := currState[id1].Longitude
				currLon2 := currState[id2].Longitude

				prevDiff := getShortestDiff(prevLon1, prevLon2)
				currDiff := getShortestDiff(currLon1, currLon2)

				for _, asp := range MajorAspects {
					if aspectCrossed(prevLon1, prevLon2, currLon1, currLon2, asp.Angle) {
						if math.Abs(prevDiff-asp.Angle) <= asp.Orb || math.Abs(currDiff-asp.Angle) <= asp.Orb {
							exactTime, _ := findExactAspect(id1, id2, asp.Name, currentTime.Add(-step), currentTime)

							events = append(events, CalendarEvent{
								Date:    exactTime.In(loc).Format("2006-01-02T15:04"),
								Type:    "exA",
								Planets: []string{strconv.Itoa(id1), strconv.Itoa(id2)},
								Aspect:  strconv.Itoa(int(asp.Angle)),
								Degrees: []int{int(currLon1), int(currLon2)},
							})
						}
					}
				}
			}
		}

		// Обновляем состояние для следующего шага
		for _, id := range planetIDs {
			prevState[id] = currState[id]
		}
	}

	// 2. Лунный цикл (шаг 1 час): лунные дни, аспекты Луны и Луна без курса
	moonStep := 1 * time.Hour
	moonStart := start.Add(-48 * time.Hour)
	tjdUtMoon := dateToJulian(moonStart)
	var xxMoon, xxSun [6]C.double

	C.swe_calc_ut(tjdUtMoon, C.int(1), C.SEFLG_SWIEPH, (*C.double)(unsafe.Pointer(&xxMoon)), (*C.char)(unsafe.Pointer(&tmpSerr)))
	moonLon := float64(xxMoon[0])

	C.swe_calc_ut(tjdUtMoon, C.int(0), C.SEFLG_SWIEPH, (*C.double)(unsafe.Pointer(&xxSun)), (*C.char)(unsafe.Pointer(&tmpSerr)))
	sunLon := float64(xxSun[0])

	diff := moonLon - sunLon
	for diff < 0 {
		diff += 360
	}
	for diff >= 360 {
		diff -= 360
	}

	prevLunarDay := int(diff/12) + 1
	prevMoonSign := int(moonLon / 30.0)

	var lastAspectTime time.Time
	var prevMoonState float64 = moonLon
	prevPlanetsState := make(map[int]float64)

	// Инициализируем координаты планет для лунного цикла
	for _, id := range planetIDs {
		if id == 1 {
			continue
		}
		var xxPlanet [6]C.double
		C.swe_calc_ut(tjdUtMoon, C.int(id), C.SEFLG_SWIEPH, (*C.double)(unsafe.Pointer(&xxPlanet)), (*C.char)(unsafe.Pointer(&tmpSerr)))
		prevPlanetsState[id] = float64(xxPlanet[0])
	}

	for currentTime := moonStart.Add(moonStep); !currentTime.After(end); currentTime = currentTime.Add(moonStep) {
		tjdUt = dateToJulian(currentTime)

		C.swe_calc_ut(tjdUt, C.int(1), C.SEFLG_SWIEPH, (*C.double)(unsafe.Pointer(&xxMoon)), (*C.char)(unsafe.Pointer(&tmpSerr)))
		moonLon = float64(xxMoon[0])

		C.swe_calc_ut(tjdUt, C.int(0), C.SEFLG_SWIEPH, (*C.double)(unsafe.Pointer(&xxSun)), (*C.char)(unsafe.Pointer(&tmpSerr)))
		sunLon = float64(xxSun[0])

		diff = moonLon - sunLon
		for diff < 0 {
			diff += 360
		}
		for diff >= 360 {
			diff -= 360
		}

		// Лунные дни
		currentLunarDay := int(diff/12) + 1
		if currentLunarDay != prevLunarDay {
			targetDay := currentLunarDay
			if prevLunarDay == 30 && currentLunarDay == 1 {
				targetDay = 1
			}
			exactTime, _ := findExactLunarDay(targetDay, currentTime.Add(-moonStep), currentTime)
			if !exactTime.Before(start) {
				lunarDayVal := new(int)
				*lunarDayVal = currentLunarDay
				events = append(events, CalendarEvent{
					Date: exactTime.In(loc).Format("2006-01-02T15:04"),
					Type: "lunD",
					Sign: lunarDayVal,
				})
			}
			prevLunarDay = currentLunarDay
		}

		currentMoonSign := int(moonLon / 30.0)
		currPlanetsState := make(map[int]float64)

		// Рассчитываем и проверяем аспекты Луны к планетам по пересечению угла
		for _, id := range planetIDs {
			if id == 1 {
				continue
			}

			var xxPlanet [6]C.double
			C.swe_calc_ut(tjdUt, C.int(id), C.SEFLG_SWIEPH, (*C.double)(unsafe.Pointer(&xxPlanet)), (*C.char)(unsafe.Pointer(&tmpSerr)))
			planetLon := float64(xxPlanet[0])
			currPlanetsState[id] = planetLon

			prevDiff := getShortestDiff(prevMoonState, prevPlanetsState[id])
			currDiff := getShortestDiff(moonLon, planetLon)

			for _, asp := range MajorAspects {
				// Проверяем пересечение точного угла аспекта Луны с планетой
				if aspectCrossed(prevMoonState, prevPlanetsState[id], moonLon, planetLon, asp.Angle) {
					if math.Abs(prevDiff-asp.Angle) <= asp.Orb || math.Abs(currDiff-asp.Angle) <= asp.Orb {
						exactAspTime, _ := findExactAspect(1, id, asp.Name, currentTime.Add(-moonStep), currentTime)

						// Записываем аспект Луны в события (если он попадает в диапазон вывода)
						if !exactAspTime.Before(start) {
							events = append(events, CalendarEvent{
								Date:    exactAspTime.In(loc).Format("2006-01-02T15:04"),
								Type:    "exA",
								Planets: []string{"1", strconv.Itoa(id)},
								Aspect:  strconv.Itoa(int(asp.Angle)),
								Degrees: []int{int(moonLon), int(planetLon)},
							})
						}

						// Сохраняем время аспекта для расчета Луны без курса
						if exactAspTime.After(lastAspectTime) {
							lastAspectTime = exactAspTime
						}
					}
				}
			}
		}

		// Луна без курса (проверяем ингрессию Луны)
		if currentMoonSign != prevMoonSign {
			exactSignTime, _ := findExactSignChange(1, currentMoonSign, currentTime.Add(-moonStep), currentTime)

			if !lastAspectTime.IsZero() && lastAspectTime.Before(exactSignTime) {
				if !exactSignTime.Before(start) {
					events = append(events, CalendarEvent{
						Date:       lastAspectTime.In(loc).Format("2006-01-02T15:04"),
						Type:       "noC",
						Planets:    []string{"1"},
						ChangeSign: exactSignTime.In(loc).Format("2006-01-02T15:04"),
					})
				}
			}

			prevMoonSign = currentMoonSign
			lastAspectTime = time.Time{}
		}

		// Обновляем состояния для следующего шага лунного цикла
		prevMoonState = moonLon
		for id, val := range currPlanetsState {
			prevPlanetsState[id] = val
		}
	}
	slices.SortFunc(events, func(a, b CalendarEvent) int {
		if a.Date < b.Date {
			return -1
		}
		if a.Date > b.Date {
			return 1
		}
		return 0
	})
	return &AstroResult{
		Type:      "calendar",
		Timestamp: start,
		Events:    events,
	}, nil
}
