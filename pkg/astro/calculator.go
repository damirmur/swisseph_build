package astro

/*
#cgo CFLAGS: -I${SRCDIR}/../../../swisseph_build/include
#cgo LDFLAGS: -L${SRCDIR}/../../../swisseph_build/lib -lswe -lm
#include "swephexp.h"
*/
import "C"

import (
	"context"
	"runtime"
	"sort"
	"time"
	"unsafe"
)

type Calculator struct {
	ephePath string
	cPath    *C.char
	locked   bool
}

func NewCalculator(ephePath string) *Calculator {
	// Привязываем горутину к текущему OS-потоку на всё время жизни калькулятора.
	// Swiss Ephemeris (предсобранная libswe.a) хранит путь к эфемеридам и файловые
	// дескрипторы в состоянии, не переживающем миграцию горутины между потоками:
	// Go переносит горутину на другой поток прямо посреди расчёта (видно в strace
	// по смене TID), после чего вызовы swe идут по путям по умолчанию и молча
	// падают в менее точную встроенную теорию Moshier — отсюда плавающие ±1 мин
	// (изредка больше) в событиях на границе минуты от прогона к прогону тем же
	// бинарником. Close снимает привязку, вызывать его обязательно (везде defer).
	runtime.LockOSThread()
	cPath := C.CString(ephePath)
	C.swe_set_ephe_path(cPath)
	return &Calculator{
		ephePath: ephePath,
		cPath:    cPath,
		locked:   true,
	}
}

func (c *Calculator) Close() {
	if c.cPath != nil {
		C.free(unsafe.Pointer(c.cPath))
		c.cPath = nil
	}
	C.swe_close()
	if c.locked {
		c.locked = false
		runtime.UnlockOSThread()
	}
}

func GetPlanetName(id int) string {
	names := map[int]string{
		0: "Sun", 1: "Moon", 2: "Mercury", 3: "Venus", 4: "Mars",
		5: "Jupiter", 6: "Saturn", 7: "Uranus", 8: "Neptune", 9: "Pluto",
		10: "Mean Node", 12: "Lilith",
	}
	return names[id]
}

// GetPlanetIDs возвращает список ID планет для расчетов
func GetPlanetIDs() []int {
	return []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12}
}

// calcPlanet рассчитывает положение планеты (без дома) на заданный юлианский момент.
func calcPlanet(id int, tjdUt C.double, xx *[6]C.double, serr *[256]C.char) (Position, bool) {
	if C.swe_calc_ut(tjdUt, C.int(id), C.SEFLG_SWIEPH|C.SEFLG_SPEED, (*C.double)(unsafe.Pointer(xx)), (*C.char)(unsafe.Pointer(serr))) < 0 {
		return Position{}, false
	}
	return Position{
		ID:        id,
		Name:      GetPlanetName(id),
		Longitude: float64(xx[0]),
		Latitude:  float64(xx[1]),
		Speed:     float64(xx[3]),
	}, true
}

// calcPlanetWithHouse рассчитывает положение планеты с номером её дома.
func calcPlanetWithHouse(id int, tjdUt, lat, armc, eps C.double, hsys C.int, xx *[6]C.double, serr *[256]C.char) (Position, bool) {
	p, ok := calcPlanet(id, tjdUt, xx, serr)
	if !ok {
		return Position{}, false
	}
	p.House = int(C.swe_house_pos(armc, lat, eps, hsys, (*C.double)(unsafe.Pointer(xx)), (*C.char)(unsafe.Pointer(serr))))
	return p, true
}

// buildAspects находит все пары мажорных аспектов внутри списка планет и сортирует их.
func buildAspects(planets []Position) []Aspect {
	var aspectsList []Aspect
	for i := 0; i < len(planets); i++ {
		for j := i + 1; j < len(planets); j++ {
			p1 := planets[i]
			p2 := planets[j]

			if name, _, exactDiff, isAspect := CalculateAspect(p1.Longitude, p2.Longitude); isAspect {
				aspectsList = append(aspectsList, Aspect{
					Planet1ID: p1.ID,
					Planet2ID: p2.ID,
					Planet1:   p1.Name,
					Planet2:   p2.Name,
					Type:      name,
					Degree:    exactDiff,
					Orb:       exactDiff,
				})
			}
		}
	}

	sort.Slice(aspectsList, func(i, j int) bool {
		if aspectsList[i].Planet1ID == aspectsList[j].Planet1ID {
			return aspectsList[i].Planet2ID < aspectsList[j].Planet2ID
		}
		return aspectsList[i].Planet1ID < aspectsList[j].Planet1ID
	})
	return aspectsList
}

// ComputeNatal выполняет натальный расчет
func (c *Calculator) ComputeNatal(ctx context.Context, t time.Time, lat, lon float64, hsys string) (*AstroResult, error) {
	var dret [2]C.double
	C.swe_utc_to_jd(
		C.int(t.Year()), C.int(int(t.Month())), C.int(t.Day()),
		C.int(t.Hour()), C.int(t.Minute()), C.double(t.Second()),
		C.SE_GREG_CAL,
		&dret[0],
		nil,
	)
	tjdUt := dret[1]

	var houses [13]C.double
	var ascmc [10]C.double
	var serr [256]C.char

	cHsys := C.int(hsys[0])
	C.swe_houses(tjdUt, C.double(lat), C.double(lon), cHsys, (*C.double)(unsafe.Pointer(&houses)), (*C.double)(unsafe.Pointer(&ascmc)))

	var housesList []House
	for i := 1; i <= 12; i++ {
		housesList = append(housesList, House{
			Number:    i,
			Longitude: float64(houses[i]),
		})
	}

	armc := ascmc[2]
	var eps [6]C.double

	C.swe_calc_ut(tjdUt, C.SE_ECL_NUT, C.SEFLG_SWIEPH, (*C.double)(unsafe.Pointer(&eps)), (*C.char)(unsafe.Pointer(&serr)))

	var planetsList []Position
	var xx [6]C.double

	for _, id := range GetPlanetIDs() {
		if p, ok := calcPlanetWithHouse(id, tjdUt, C.double(lat), armc, eps[0], cHsys, &xx, &serr); ok {
			planetsList = append(planetsList, p)
		}
	}

	var aspectsList []Aspect
	aspectsList = buildAspects(planetsList)

	return &AstroResult{
		Type:      "natal",
		Timestamp: t,
		Planets:   planetsList,
		Houses:    housesList,
		Aspects:   aspectsList,
	}, nil
}

// ComputeSynastry выполняет расчет совместимости
func (c *Calculator) ComputeSynastry(ctx context.Context, t1, t2 time.Time, lat1, lon1 float64, hsys string) (*AstroResult, error) {
	var dret1, dret2 [2]C.double
	C.swe_utc_to_jd(C.int(t1.Year()), C.int(int(t1.Month())), C.int(t1.Day()), C.int(t1.Hour()), C.int(t1.Minute()), C.double(t1.Second()), C.SE_GREG_CAL, &dret1[0], nil)
	C.swe_utc_to_jd(C.int(t2.Year()), C.int(int(t2.Month())), C.int(t2.Day()), C.int(t2.Hour()), C.int(t2.Minute()), C.double(t2.Second()), C.SE_GREG_CAL, &dret2[0], nil)
	tjdUt1 := dret1[1]
	tjdUt2 := dret2[1]

	var houses [13]C.double
	var ascmc [10]C.double
	var serr [256]C.char
	cHsys := C.int(hsys[0])
	C.swe_houses(tjdUt1, C.double(lat1), C.double(lon1), cHsys, (*C.double)(unsafe.Pointer(&houses)), (*C.double)(unsafe.Pointer(&ascmc)))

	var housesList []House
	for i := 1; i <= 12; i++ {
		housesList = append(housesList, House{Number: i, Longitude: float64(houses[i])})
	}

	armc1 := ascmc[2]
	var eps1 [6]C.double
	C.swe_calc_ut(tjdUt1, C.SE_ECL_NUT, C.SEFLG_SWIEPH, (*C.double)(unsafe.Pointer(&eps1)), (*C.char)(unsafe.Pointer(&serr)))

	var planetsA []Position
	var xx [6]C.double

	for _, id := range GetPlanetIDs() {
		if p, ok := calcPlanetWithHouse(id, tjdUt1, C.double(lat1), armc1, eps1[0], cHsys, &xx, &serr); ok {
			planetsA = append(planetsA, p)
		}
	}

	var planetsB []Position
	for _, id := range GetPlanetIDs() {
		if p, ok := calcPlanetWithHouse(id, tjdUt2, C.double(lat1), armc1, eps1[0], cHsys, &xx, &serr); ok {
			planetsB = append(planetsB, p)
		}
	}

	var aspectsList []Aspect
	for _, pB := range planetsB {
		for _, pA := range planetsA {
			if name, _, exactDiff, isAspect := CalculateAspect(pB.Longitude, pA.Longitude); isAspect {
				aspectsList = append(aspectsList, Aspect{
					Planet1ID: pB.ID,
					Planet2ID: pA.ID,
					Planet1:   pB.Name + " (Transit)",
					Planet2:   pA.Name + " (Natal)",
					Type:      name,
					Degree:    exactDiff,
					Orb:       exactDiff,
				})
			}
		}
	}

	sort.Slice(aspectsList, func(i, j int) bool {
		if aspectsList[i].Planet1ID == aspectsList[j].Planet1ID {
			return aspectsList[i].Planet2ID < aspectsList[j].Planet2ID
		}
		return aspectsList[i].Planet1ID < aspectsList[j].Planet1ID
	})

	return &AstroResult{
		Type:      "synastry",
		Timestamp: t2,
		Planets:   planetsA,
		Houses:    housesList,
		PlanetsB:  planetsB,
		HousesB:   housesList,
		Aspects:   aspectsList,
	}, nil
}

// ComputePeriod выполняет расчет движения за период
func (c *Calculator) ComputePeriod(ctx context.Context, start, end time.Time, stepHours int) (*AstroResult, error) {
	if stepHours <= 0 {
		stepHours = 24
	}

	var slices []TimeSlice

	for currentTime := start; !currentTime.After(end); currentTime = currentTime.Add(time.Duration(stepHours) * time.Hour) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var dret [2]C.double
		C.swe_utc_to_jd(C.int(currentTime.Year()), C.int(int(currentTime.Month())), C.int(currentTime.Day()), C.int(currentTime.Hour()), C.int(currentTime.Minute()), C.double(currentTime.Second()), C.SE_GREG_CAL, &dret[0], nil)
		tjdUt := dret[1]

		var planets []Position
		var xx [6]C.double
		var serr [256]C.char

		for _, id := range GetPlanetIDs() {
			if p, ok := calcPlanet(id, tjdUt, &xx, &serr); ok {
				planets = append(planets, p)
			}
		}

		var aspects []Aspect
		aspects = buildAspects(planets)

		slices = append(slices, TimeSlice{
			Timestamp: currentTime,
			Planets:   planets,
			Aspects:   aspects,
		})
	}

	return &AstroResult{
		Type:      "period",
		Timestamp: start,
		Slices:    slices,
	}, nil
}

func dateToJulian(t time.Time) C.double {
	var dret [2]C.double
	C.swe_utc_to_jd(
		C.int(t.Year()), C.int(int(t.Month())), C.int(t.Day()),
		C.int(t.Hour()), C.int(t.Minute()), C.double(t.Second()),
		C.SE_GREG_CAL,
		&dret[0],
		nil,
	)
	return dret[1] // dret[0] is ET, dret[1] is UT
}
