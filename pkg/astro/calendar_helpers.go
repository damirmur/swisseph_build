package astro

/*
#cgo CFLAGS: -I${SRCDIR}/../../../swisseph_build/include
#cgo LDFLAGS: -L${SRCDIR}/../../../swisseph_build/lib -lswe -lm
#include "swephexp.h"
*/
import "C"
import (
	"math"
	"time"
	"unsafe"
)

const phi = 1.618033988749895
const resphi = 2 - phi // 0.381966011250105

// getPlanetLon returns longitude of a planet at given time
func getPlanetLon(id int, t time.Time) float64 {
	tjdUt := dateToJulian(t)
	var xx [6]C.double
	var serr [256]C.char
	C.swe_calc_ut(tjdUt, C.int(id), C.SEFLG_SWIEPH, (*C.double)(unsafe.Pointer(&xx)), (*C.char)(unsafe.Pointer(&serr)))
	return float64(xx[0])
}

// goldenSectionSearch minimizes function f on interval [a, b]
func goldenSectionSearch(a, b time.Time, f func(time.Time) float64) (time.Time, error) {
	tA := a.UnixNano()
	tB := b.UnixNano()

	c := tA + int64(resphi*float64(tB-tA))
	d := tB - int64(resphi*float64(tB-tA))

	fc := f(time.Unix(0, c).UTC())
	fd := f(time.Unix(0, d).UTC())

	for i := 0; i < 20; i++ {
		if fc < fd {
			tB = d
			d = c
			fd = fc
			c = tA + int64(resphi*float64(tB-tA))
			fc = f(time.Unix(0, c).UTC())
		} else {
			tA = c
			c = d
			fc = fd
			d = tB - int64(resphi*float64(tB-tA))
			fd = f(time.Unix(0, d).UTC())
		}
	}

	return time.Unix(0, (tA+tB)/2).UTC(), nil
}

func findExactSignChange(id int, targetSign int, start, end time.Time) (time.Time, error) {
	targetLon := float64(targetSign * 30)

	f := func(t time.Time) float64 {
		lon := getPlanetLon(id, t)
		diff := lon - targetLon
		if diff < -180 {
			diff += 360
		} else if diff > 180 {
			diff -= 360
		}
		return math.Abs(diff)
	}

	return goldenSectionSearch(start, end, f)
}

func findExactAspect(id1, id2 int, aspectName string, start, end time.Time) (time.Time, error) {
	targetAngle := aspectAngle(aspectName)

	f := func(t time.Time) float64 {
		diff := getShortestDiff(getPlanetLon(id1, t), getPlanetLon(id2, t)) - targetAngle
		return math.Abs(diff)
	}

	return goldenSectionSearch(start, end, f)
}

func findExactLunarDay(targetDay int, start, end time.Time) (time.Time, error) {
	targetAngle := float64((targetDay - 1) * 12)

	f := func(t time.Time) float64 {
		lonMoon := getPlanetLon(1, t)
		lonSun := getPlanetLon(0, t)
		diff := lonMoon - lonSun
		for diff < 0 {
			diff += 360
		}
		for diff >= 360 {
			diff -= 360
		}

		err := diff - targetAngle
		if err > 180 {
			err -= 360
		}
		if err < -180 {
			err += 360
		}
		return math.Abs(err)
	}

	return goldenSectionSearch(start, end, f)
}

// findExactStationTime находит точное время разворота планеты (когда скорость по долготе = 0)
func findExactStationTime(planetID int, startTime, endTime time.Time) (time.Time, error) {
	low := startTime
	high := endTime
	var midXX [6]C.double
	var lowXX [6]C.double
	var tmpSerr [256]C.char

	// 10 итераций деления отрезка пополам дают точность ~1.4 минуты
	for i := 0; i < 10; i++ {
		mid := low.Add(high.Sub(low) / 2)

		C.swe_calc_ut(dateToJulian(mid), C.int(planetID), C.SEFLG_SWIEPH|C.SEFLG_SPEED, (*C.double)(unsafe.Pointer(&midXX)), (*C.char)(unsafe.Pointer(&tmpSerr)))
		C.swe_calc_ut(dateToJulian(low), C.int(planetID), C.SEFLG_SWIEPH|C.SEFLG_SPEED, (*C.double)(unsafe.Pointer(&lowXX)), (*C.char)(unsafe.Pointer(&tmpSerr)))

		midSpeed := float64(midXX[3])
		lowSpeed := float64(lowXX[3])

		// Если знаки скорости совпадают, значит ноль лежит в правой половине
		if (lowSpeed > 0 && midSpeed > 0) || (lowSpeed < 0 && midSpeed < 0) {
			low = mid
		} else {
			high = mid
		}
	}
	return low.Add(high.Sub(low) / 2), nil
}
