// pkg/astro/returns.go
//
// Возвраты планет (соляры/лунары): моменты, когда транзитная планета
// возвращается на натальную долготу. Предыдущий/последующий — относительно
// опорной транзитной даты (ref).
package astro

/*
#cgo CFLAGS: -I${SRCDIR}/../../../swisseph_build/include
#cgo LDFLAGS: -L${SRCDIR}/../../../swisseph_build/lib -lswe -lm
#include "swephexp.h"
*/
import "C"

import (
	"fmt"
	"time"
	"unsafe"
)

// Окна поиска предыдущего возврата: с запасом покрывают период обращения.
const (
	// Солнце возвращается ~раз в год — ищем назад с запасом 400 дней.
	solarSearchBackDays = 400
	// Луна возвращается ~раз в 27.3 дня — ищем назад с запасом 40 дней.
	lunarSearchBackDays = 40
)

// ReturnPair — предыдущий и последующий возвраты относительно ref.
type ReturnPair struct {
	Prev time.Time
	Next time.Time
}

// ReturnsResult — все четыре момента для опорной даты.
type ReturnsResult struct {
	Solar ReturnPair
	Lunar ReturnPair
	// Натальные долготы, на которые возвращались (для отладки/подписей).
	NatalSunLon  float64
	NatalMoonLon float64
}

// ReturnsSchema — схема самоописываемого ответа /api/v1/returns.
const ReturnsSchema = "astro3d.returns/v1"

// ReturnMoment — пара моментов возврата в ISO-8601 (с секундами, в зоне ref).
type ReturnMoment struct {
	Prev string `json:"prev"`
	Next string `json:"next"`
}

// ReturnsData — блок data самоописываемого ответа /api/v1/returns.
type ReturnsData struct {
	Solar ReturnMoment `json:"solar"`
	Lunar ReturnMoment `json:"lunar"`
	Natal ReturnsNatal `json:"natal"`
}

// ReturnsNatal — натальные долготы возвращающихся планет.
type ReturnsNatal struct {
	SunLon  float64 `json:"sun_lon"`
	MoonLon float64 `json:"moon_lon"`
}

// ReturnsGeo — гео/временная привязка натала и опорной даты.
type ReturnsGeo struct {
	Natal GeoInfo `json:"natal"`
	Ref   GeoInfo `json:"ref"`
}

// ToReturnsData упаковывает результат в компактный блок data,
// форматируя моменты в зоне ref (ISO-8601 с секундами).
func (r *ReturnsResult) ToReturnsData(loc *time.Location) ReturnsData {
	if loc == nil {
		loc = time.UTC
	}
	const layout = "2006-01-02T15:04:05Z07:00"
	return ReturnsData{
		Solar: ReturnMoment{
			Prev: r.Solar.Prev.In(loc).Format(layout),
			Next: r.Solar.Next.In(loc).Format(layout),
		},
		Lunar: ReturnMoment{
			Prev: r.Lunar.Prev.In(loc).Format(layout),
			Next: r.Lunar.Next.In(loc).Format(layout),
		},
		Natal: ReturnsNatal{SunLon: r.NatalSunLon, MoonLon: r.NatalMoonLon},
	}
}

// NewReturnsMeta — meta самоописываемого ответа /api/v1/returns.
func NewReturnsMeta() Meta {
	return NewAstroMeta(map[string]MetaField{
		"solar": {
			Type:        "object",
			Title:       "Соляры (возвраты Солнца)",
			Description: "Объект {prev, next} — предыдущий/последующий моменты возврата Солнца на натальную долготу (ISO-8601 с секундами, в зоне ref)",
		},
		"lunar": {
			Type:        "object",
			Title:       "Лунары (возвраты Луны)",
			Description: "Объект {prev, next} — предыдущий/последующий моменты возврата Луны на натальную долготу (ISO-8601 с секундами, в зоне ref)",
		},
		"natal": {
			Type:        "object",
			Title:       "Натальные долготы",
			Description: "Объект {sun_lon, moon_lon} — долготы, на которые возвращались (градусы 0-360)",
		},
	}, map[string]string{
		"city":      "Город натала для геокодирования (альтернатива lat/lon)",
		"lat":       "Широта натала (десятичные градусы)",
		"lon":       "Долгота натала (десятичные градусы)",
		"date":      "Дата натала (YYYY-MM-DD)",
		"time":      "Время натала (HH:MM)",
		"hsys":      "Система домов (по умолчанию P; на моменты возвратов не влияет)",
		"ref_date":  "Опорная транзитная дата (YYYY-MM-DD, обязательный)",
		"ref_time":  "Опорное транзитное время (HH:MM, по умолчанию 12:00)",
		"ref_city":  "Город опорной даты (по умолчанию — город натала)",
		"ref_lat":   "Широта опорной даты (альтернатива ref_city)",
		"ref_lon":   "Долгота опорной даты (альтернатива ref_city)",
	})
}

// jdToUTC переводит юлианский день UT в UTC-момент (с секундами).
func jdToUTC(jd float64) time.Time {
	var year, mon, day, hour, min C.int
	var dsec C.double
	C.swe_jdut1_to_utc(C.double(jd), C.SE_GREG_CAL,
		&year, &mon, &day, &hour, &min, &dsec)
	sec := int(dsec)
	nsec := int((float64(dsec) - float64(sec)) * 1e9)
	return time.Date(int(year), time.Month(int(mon)), int(day),
		int(hour), int(min), sec, nsec, time.UTC)
}

// natalPlanetLon — геоцентрическая долгота планеты на натальный момент.
func natalPlanetLon(id int, natal time.Time) (float64, error) {
	tjdUt := dateToJulian(natal)
	var xx [6]C.double
	var serr [256]C.char
	if C.swe_calc_ut(tjdUt, C.int(id), C.SEFLG_SWIEPH,
		(*C.double)(unsafe.Pointer(&xx)), (*C.char)(unsafe.Pointer(&serr))) < 0 {
		return 0, fmt.Errorf("не удалось рассчитать планету %d на натальный момент", id)
	}
	return float64(xx[0]), nil
}

// nextCrossing — ближайший возврат (пересечение lon) строго после ref.
// isSolar выбирает swe_solcross_ut (Солнце) или swe_mooncross_ut (Луна).
func nextCrossing(lon float64, ref time.Time, isSolar bool) (time.Time, error) {
	jdRef := float64(dateToJulian(ref))
	var serr [256]C.char
	var jdRet C.double
	if isSolar {
		jdRet = C.swe_solcross_ut(C.double(lon), C.double(jdRef),
			C.SEFLG_SWIEPH, (*C.char)(unsafe.Pointer(&serr)))
	} else {
		jdRet = C.swe_mooncross_ut(C.double(lon), C.double(jdRef),
			C.SEFLG_SWIEPH, (*C.char)(unsafe.Pointer(&serr)))
	}
	if float64(jdRet) == 0 {
		return time.Time{}, fmt.Errorf("swe_*cross_ut не нашёл возврат для долготы %.4f", lon)
	}
	return jdToUTC(float64(jdRet)), nil
}

// returnPair ищет предыдущий и последующий возвраты на долготу lon
// относительно опорной даты ref. swe_*cross_ut находит только следующий
// возврат после заданного JD, поэтому для предыдущего стартуем заранее
// (backDays) и шагаем вперёд, пока пересечение не подойдёт к ref вплотную.
func returnPair(lon float64, ref time.Time, backDays int, isSolar bool) (ReturnPair, error) {
	const epsSec = 2 // ref в пределах 2с от возврата считаем точным попаданием
	var pair ReturnPair

	next, err := nextCrossing(lon, ref, isSolar)
	if err != nil {
		return pair, err
	}
	// Точное попадание: ref уже момент возврата — он и есть next,
	// а предыдущий ищем от чуть более ранней точки.
	if absDuration(next.Sub(ref)) <= epsSec*time.Second {
		prev, err := returnPairBefore(lon, ref.Add(-time.Hour), backDays, isSolar)
		if err != nil {
			return pair, err
		}
		return ReturnPair{Prev: prev, Next: next}, nil
	}
	prev, err := returnPairBefore(lon, ref, backDays, isSolar)
	if err != nil {
		return pair, err
	}
	return ReturnPair{Prev: prev, Next: next}, nil
}

// returnPairBefore — последний возврат строго до ref.
func returnPairBefore(lon float64, ref time.Time, backDays int, isSolar bool) (time.Time, error) {
	start := ref.AddDate(0, 0, -backDays)
	cand, err := nextCrossing(lon, start, isSolar)
	if err != nil {
		return time.Time{}, err
	}
	// Шагаем вперёд, пока следующее пересечение остаётся до ref.
	for {
		if !cand.Before(ref) {
			// Первый кандидат уже за ref — значит окно слишком узкое.
			return time.Time{}, fmt.Errorf("окно поиска %d дн. мало для возврата (долгота %.4f)", backDays, lon)
		}
		nxt, err := nextCrossing(lon, cand.Add(time.Hour), isSolar)
		if err != nil {
			return time.Time{}, err
		}
		if !nxt.Before(ref) {
			return cand, nil
		}
		cand = nxt
	}
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

// ComputeReturns рассчитывает предыдущий/последующий соляры и лунары
// относительно опорной даты ref. Долготы берутся с натального момента natal.
// Вызывать между NewCalculator и Close (те же требования к потоку, что и
// остальные методы Calculator).
func (c *Calculator) ComputeReturns(natal, ref time.Time) (*ReturnsResult, error) {
	sunLon, err := natalPlanetLon(0, natal)
	if err != nil {
		return nil, err
	}
	moonLon, err := natalPlanetLon(1, natal)
	if err != nil {
		return nil, err
	}
	solar, err := returnPair(sunLon, ref, solarSearchBackDays, true)
	if err != nil {
		return nil, fmt.Errorf("соляр: %w", err)
	}
	lunar, err := returnPair(moonLon, ref, lunarSearchBackDays, false)
	if err != nil {
		return nil, fmt.Errorf("лунар: %w", err)
	}
	return &ReturnsResult{
		Solar:        solar,
		Lunar:        lunar,
		NatalSunLon:  sunLon,
		NatalMoonLon: moonLon,
	}, nil
}
