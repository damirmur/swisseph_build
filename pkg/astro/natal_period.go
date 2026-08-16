// pkg/astro/natal_period.go
package astro

/*
#cgo CFLAGS: -I${SRCDIR}/../../../swisseph_build/include
#cgo LDFLAGS: -L${SRCDIR}/../../../swisseph_build/lib -lswe -lm
#include "swephexp.h"
*/
import "C"

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"time"
)

// TransitPoint — положение транзитной планеты относительно натальной карты.
// Сериализуется как [longitude, speed, house] (house — номер дома натала 1-12).
type TransitPoint struct {
	Longitude float64
	Speed     float64
	House     int
}

// MarshalJSON реализует форму [longitude, speed, house].
func (p TransitPoint) MarshalJSON() ([]byte, error) {
	return json.Marshal([]interface{}{p.Longitude, p.Speed, p.House})
}

// TransitSeriesData — ряды транзитов по срезам периода.
type TransitSeriesData struct {
	Date    []string                `json:"date"`
	Planets map[string][]TransitPoint `json:"planets"` // id -> [ [lon,speed,house], ... ] по срезам
}

// TransitAspectEvent — событие аспекта транзит->натал за период.
// kind: "enter" (вход в орб), "exact" (точный/ближайший к точному),
// "leave" (выход из орба).
type TransitAspectEvent struct {
	Date      string  `json:"date"`
	TransitID int     `json:"transit_id"`
	NatalID   int     `json:"natal_id"`
	Type      string  `json:"type"`
	Angle     float64 `json:"angle"`
	Orb       float64 `json:"orb"`
	Kind      string  `json:"kind"`
}

// NatalPeriodData — блок data самоописываемого ответа /api/v1/natal-period.
type NatalPeriodData struct {
	Natal    NatalData            `json:"natal"`
	Transits TransitSeriesData    `json:"transits"`
	Aspects  []TransitAspectEvent `json:"aspects"`
}

// NatalPeriodGeo — гео/временная привязка натала и периода.
type NatalPeriodGeo struct {
	Natal  GeoInfo `json:"natal"`
	Period GeoInfo `json:"period"`
}

// DefaultTransitPlanets — планеты транзитов по умолчанию (без 11 North Node и 13 Osc. Apogee).
var DefaultTransitPlanets = []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12}

// HouseOfLongitude возвращает номер дома (1-12) для долготы lon по заданным куспидам.
// Куспиды заданы в порядке домов 1..12 (индексы 0..11).
func HouseOfLongitude(lon float64, cusps []float64) int {
	if len(cusps) < 12 {
		return 0
	}
	for h := 1; h <= 12; h++ {
		start := cusps[h-1]
		end := cusps[h%12] // дом 12 заканчивается на куспиде 1
		if inClockwiseArc(lon, start, end) {
			return h
		}
	}
	return 0
}

func inClockwiseArc(lon, start, end float64) bool {
	s := math.Mod(start, 360)
	if s < 0 {
		s += 360
	}
	e := math.Mod(end, 360)
	if e < 0 {
		e += 360
	}
	l := math.Mod(lon, 360)
	if l < 0 {
		l += 360
	}
	if e <= s {
		e += 360
	}
	if l < s {
		l += 360
	}
	return l < e
}

// interpLon линейно интерполирует долготу между a и b по доле f,
// учитывая цикличность (кратчайший путь по кругу).
func interpLon(a, b, f float64) float64 {
	diff := math.Mod(b-a, 360)
	if diff > 180 {
		diff -= 360
	} else if diff < -180 {
		diff += 360
	}
	return math.Mod(a+diff*f, 360)
}

// natalPeriodPairKey — ключ состояния аспекта (транзитная планета, натальная планета, угол).
type natalPeriodPairKey struct {
	Transit int
	Natal   int
	Angle   float64
}

type pairState struct {
	inside    bool
	prevTime  time.Time
	prevLonT  float64
	prevD     float64
	minD      float64
	minTime   time.Time
	minLonT   float64
	hasMin    bool
	transitID int
	natalID   int
	cfg       AspectConfig
}

// ComputeNatalPeriod считает натальную карту и транзиты за период,
// определяя для каждой транзитной планеты её положение в домах натала,
// а также события аспектов транзит->натал (вход/точный/выход) внутри орба maxOrb.
func (c *Calculator) ComputeNatalPeriod(ctx context.Context, natalTime time.Time, lat, lon float64, hsys string,
	start, end time.Time, stepHours int, transitIDs []int, maxOrb float64, aspectAngles []float64, outLoc *time.Location) (*NatalPeriodData, error) {

	if stepHours <= 0 {
		stepHours = 24
	}

	// 1. Натальная карта (планеты + дома) — фиксированная база для домов транзитов.
	natal, err := c.ComputeNatal(ctx, natalTime, lat, lon, hsys)
	if err != nil {
		return nil, err
	}
	cusps := make([]float64, 0, len(natal.Houses))
	for _, h := range natal.Houses {
		cusps = append(cusps, h.Longitude)
	}
	natalLons := make([]float64, len(natal.Planets))
	for i, p := range natal.Planets {
		natalLons[i] = p.Longitude
	}
	natalIDs := make([]int, len(natal.Planets))
	for i, p := range natal.Planets {
		natalIDs[i] = p.ID
	}

	cfgs := AspectConfigs(aspectAngles, maxOrb)

	// 2. Проход по срезам периода.
	series := TransitSeriesData{
		Date:    []string{},
		Planets: map[string][]TransitPoint{},
	}
	for _, id := range transitIDs {
		series.Planets[strconv.Itoa(id)] = []TransitPoint{}
	}

	states := map[natalPeriodPairKey]*pairState{}
	var events []TransitAspectEvent

	sliceIdx := 0
	for currentTime := start; !currentTime.After(end); currentTime = currentTime.Add(time.Duration(stepHours) * time.Hour) {
		// Позиции транзитных планет на этот срез.
		var xx [6]C.double
		var serr [256]C.char
		transitLons := make([]float64, len(transitIDs))
		transitSpeeds := make([]float64, len(transitIDs))
		for i, id := range transitIDs {
			if p, ok := calcPlanet(id, dateToJulian(currentTime), &xx, &serr); ok {
				transitLons[i] = p.Longitude
				transitSpeeds[i] = p.Speed
			}
		}

		// Заполняем ряд транзитов (долгота, скорость, дом натала).
		series.Date = append(series.Date, currentTime.In(outLoc).Format(time.RFC3339))
		for i, id := range transitIDs {
			house := HouseOfLongitude(transitLons[i], cusps)
			series.Planets[strconv.Itoa(id)] = append(series.Planets[strconv.Itoa(id)],
				TransitPoint{Longitude: transitLons[i], Speed: transitSpeeds[i], House: house})
		}

		// Обнаружение аспектов транзит->натал по срезам.
		for ti, tid := range transitIDs {
			lt := transitLons[ti]
			for ni, nid := range natalIDs {
				ln := natalLons[ni]
				diff := getShortestDiff(lt, ln)
				for _, cfg := range cfgs {
					d := math.Abs(diff - cfg.Angle)
					key := natalPeriodPairKey{Transit: tid, Natal: nid, Angle: cfg.Angle}
					st := states[key]
					if st == nil {
						st = &pairState{transitID: tid, natalID: nid, cfg: cfg}
						states[key] = st
					}
					if sliceIdx == 0 {
						st.prevTime = currentTime
						st.prevLonT = lt
						st.prevD = d
						st.inside = d <= maxOrb
						if st.inside {
							st.minD = d
							st.minTime = currentTime
							st.minLonT = lt
							st.hasMin = true
						}
						continue
					}
					insidePrev := st.inside
					insideNow := d <= maxOrb
					if insideNow && !insidePrev {
						// Вход в орб: интерполяция между prev (снаружи) и сейчас (внутри).
						f := 0.0
						if st.prevD != d {
							f = (st.prevD - maxOrb) / (st.prevD - d)
						}
						et := st.prevTime.Add(time.Duration(f * float64(currentTime.Sub(st.prevTime))))
						emitTransitEvent(&events, et, tid, nid, cfg, maxOrb, "enter", interpLon(st.prevLonT, lt, f), outLoc)
					}
					if insideNow {
						if !st.hasMin || d < st.minD {
							st.minD = d
							st.minTime = currentTime
							st.minLonT = lt
							st.hasMin = true
						}
					}
					if !insideNow && insidePrev {
						// Выход из орба: интерполяция между prev (внутри) и сейчас (снаружи).
						f := 0.0
						if d != st.prevD {
							f = (maxOrb - st.prevD) / (d - st.prevD)
						}
						xt := st.prevTime.Add(time.Duration(f * float64(currentTime.Sub(st.prevTime))))
						// Точный аспект — в момент минимума орба эпизода.
						if st.hasMin {
							emitTransitEvent(&events, st.minTime, tid, nid, cfg, st.minD, "exact", st.minLonT, outLoc)
						}
						emitTransitEvent(&events, xt, tid, nid, cfg, maxOrb, "leave", interpLon(st.prevLonT, lt, f), outLoc)
						st.inside = false
						st.hasMin = false
						st.minD = 0
					}
					st.prevD = d
					st.prevTime = currentTime
					st.prevLonT = lt
					st.inside = insideNow
				}
			}
		}
		sliceIdx++
	}

	// Для эпизодов, не закрывшихся к концу периода, фиксируем точный аспект.
	for _, st := range states {
		if st.inside && st.hasMin {
			emitTransitEvent(&events, st.minTime, st.transitID, st.natalID, st.cfg, st.minD, "exact", st.minLonT, outLoc)
		}
	}

	return &NatalPeriodData{
		Natal:    ToNatalData(natal),
		Transits: series,
		Aspects:  events,
	}, nil
}

func emitTransitEvent(events *[]TransitAspectEvent, t time.Time, transitID, natalID int, cfg AspectConfig, orb float64, kind string, lonT float64, outLoc *time.Location) {
	_ = lonT // долгота транзита в момент события (при необходимости можно добавить в структуру)
	*events = append(*events, TransitAspectEvent{
		Date:      t.In(outLoc).Format(time.RFC3339),
		TransitID: transitID,
		NatalID:   natalID,
		Type:      cfg.Name,
		Angle:     cfg.Angle,
		Orb:       orb,
		Kind:      kind,
	})
}

// NewNatalPeriodMeta — meta самоописываемого ответа /api/v1/natal-period.
func NewNatalPeriodMeta() Meta {
	return NewAstroMeta(map[string]MetaField{
		"natal": {
			Type:        "object",
			Title:       "Натальная карта",
			Description: "Объект {planets:[[id,[lon,speed]]], cusps:[12]}",
		},
		"transits": {
			Type:        "object",
			Title:       "Транзиты за период",
			Description: "Объект {date:[iso...], planets:{id:[[lon,speed,house], ...]}}",
		},
		"aspects": {
			Type:        "array",
			Title:       "События аспектов транзит->натал",
			Description: "Массив {date, transit_id, natal_id, type, angle, orb, kind}; kind: enter/exact/leave",
			Ref:         "aspects",
		},
	}, map[string]string{
		"city":     "Город натальной карты и часовой пояс периода (альтернатива lat/lon)",
		"lat":      "Широта натальной карты",
		"lon":      "Долгота натальной карты",
		"date":     "Дата/время натальной карты (ISO-8601 или Y-M-D H:M:S)",
		"time":     "Время натальной карты (HH:MM, по умолчанию 12:00)",
		"hsys":     "Система домов (по умолчанию P)",
		"tz":       "Часовой пояс IANA (необязательно)",
		"start":    "Начало периода (ISO-8601); по умолчанию — текущий момент",
		"end":      "Конец периода (ISO-8601); по умолчанию — +1 месяц от начала",
		"step":     "Шаг между срезами в часах (по умолчанию 24)",
		"planets":  "Список транзитных планет (коды через запятую); по умолчанию 0-10,12",
		"max_orb":  "Орб аспекта в градусах (по умолчанию 1)",
		"aspects":  "Фильтр углов аспектов (через запятую); по умолчанию мажорные 0,60,90,120,180",
	})
}
