// pkg/astro/astro_response.go
package astro

import (
	"encoding/json"
	"strconv"
	"time"
)

// PlanetPoint — компактное положение планеты: [id, [longitude, speed]].
// Исключает ремаппинг на фронтенде (совпадает со статическим форматом chart.out.planets).
type PlanetPoint struct {
	ID       int
	Longitude float64
	Speed    float64
}

// MarshalJSON реализует форму [id, [longitude, speed]].
func (p PlanetPoint) MarshalJSON() ([]byte, error) {
	return json.Marshal([]interface{}{p.ID, []interface{}{p.Longitude, p.Speed}})
}

func newPlanetPoint(id int, lon, speed float64) PlanetPoint {
	return PlanetPoint{ID: id, Longitude: lon, Speed: speed}
}

// NatalData — данные натальной карты в формате, совместимом со статическим chart.out.
type NatalData struct {
	Planets []PlanetPoint `json:"planets"`
	Cusps   []float64    `json:"cusps"`
	Aspects []Aspect      `json:"aspects,omitempty"`
}

// PeriodData — данные периода в формате статического chartPeriod.out.
type PeriodData struct {
	Date    []string             `json:"date"`
	Planets map[string][]float64 `json:"planets"`
	Speed   map[string][]float64 `json:"speed"`
}

// SynastryData — данные синастрии: две карты + межкартные аспекты.
type SynastryData struct {
	Chart1  NatalData `json:"chart1"`
	Chart2  NatalData `json:"chart2"`
	Aspects []Aspect  `json:"aspects"`
}

// GeoInfo — гео/временная привязка расчёта (один объект карты).
type GeoInfo struct {
	City      string  `json:"city,omitempty"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timezone  string  `json:"timezone"`
	UTCOffset float64 `json:"utc_offset"`
}

// GeoInfoPair — привязка для синастрии (две карты).
type GeoInfoPair struct {
	Chart1 GeoInfo `json:"chart1"`
	Chart2 GeoInfo `json:"chart2"`
}

// Envelope — универсальная самоописываемая огибающая ответа любого эндпоинта.
// meta — единственный источник истины для кодов и структуры; data — готовые данные.
type Envelope struct {
	Schema string      `json:"schema"`
	Meta   Meta        `json:"meta"`
	Geo    interface{} `json:"geo,omitempty"`
	Data   interface{} `json:"data"`
}

const (
	NatalSchema        = "astro3d.natal/v1"
	PeriodSchema       = "astro3d.period/v1"
	SynastrySchema     = "astro3d.synastry/v1"
	NatalPeriodSchema  = "astro3d.natal_period/v1"
)

// ToNatalData преобразует AstroResult натальной карты в компактный NatalData.
func ToNatalData(res *AstroResult) NatalData {
	planets := make([]PlanetPoint, 0, len(res.Planets))
	for _, p := range res.Planets {
		planets = append(planets, newPlanetPoint(p.ID, p.Longitude, p.Speed))
	}
	cusps := make([]float64, 0, len(res.Houses))
	for _, h := range res.Houses {
		cusps = append(cusps, h.Longitude)
	}
	return NatalData{Planets: planets, Cusps: cusps, Aspects: res.Aspects}
}

// ToPeriodData преобразует AstroResult периода в компактный PeriodData.
func ToPeriodData(res *AstroResult) PeriodData {
	dates := make([]string, 0, len(res.Slices))
	planets := map[string][]float64{}
	speeds := map[string][]float64{}
	for _, s := range res.Slices {
		dates = append(dates, s.Timestamp.Format(time.RFC3339))
		for _, p := range s.Planets {
			id := strconv.Itoa(p.ID)
			planets[id] = append(planets[id], p.Longitude)
			speeds[id] = append(speeds[id], p.Speed)
		}
	}
	return PeriodData{Date: dates, Planets: planets, Speed: speeds}
}

// ToSynastryData преобразует AstroResult синастрии в компактный SynastryData.
func ToSynastryData(res *AstroResult) SynastryData {
	chart1 := ToNatalData(res)
	chart2 := NatalData{}
	if len(res.PlanetsB) > 0 {
		planets := make([]PlanetPoint, 0, len(res.PlanetsB))
		for _, p := range res.PlanetsB {
			planets = append(planets, newPlanetPoint(p.ID, p.Longitude, p.Speed))
		}
		cusps := make([]float64, 0, len(res.HousesB))
		for _, h := range res.HousesB {
			cusps = append(cusps, h.Longitude)
		}
		chart2 = NatalData{Planets: planets, Cusps: cusps}
	}
	return SynastryData{Chart1: chart1, Chart2: chart2, Aspects: res.Aspects}
}

// NewEnvelope строит самоописываемую огибающую ответа.
func NewEnvelope(schema string, meta Meta, geo, data interface{}) Envelope {
	return Envelope{Schema: schema, Meta: meta, Geo: geo, Data: data}
}
