package astro

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Position struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	Longitude float64 `json:"longitude"` // 0-360 градусов
	Latitude  float64 `json:"latitude"`
	Speed     float64 `json:"speed"`
	House     int     `json:"house,omitempty"` // Номер дома, в котором стоит планета
}

type Aspect struct {
	Planet1ID int     `json:"planet_1_id"`
	Planet2ID int     `json:"planet_2_id"`
	Planet1   string  `json:"planet_1"`
	Planet2   string  `json:"planet_2"`
	Type      string  `json:"type"`   // Conjunction, Opposition, Trine, Square, Sextile
	Degree    float64 `json:"degree"` // Точный угол
	Orb       float64 `json:"orb"`    // Орбис
}

type House struct {
	Number    int     `json:"number"`
	Longitude float64 `json:"longitude"` // Куспид дома
}

type TimeSlice struct {
	Timestamp time.Time  `json:"timestamp"`
	Planets   []Position `json:"planets"`
	Aspects   []Aspect   `json:"aspects,omitempty"`
}

type CalendarEvent struct {
	Date       string   `json:"date"`
	Type       string   `json:"type"`
	Planets    []string `json:"planets,omitempty"`
	Sign       *int     `json:"sign,omitempty"`
	Aspect     string   `json:"aspect,omitempty"`
	Degrees    []int    `json:"degrees,omitempty"`
	ChangeSign string   `json:"change_sign,omitempty"`
}

type AstroResult struct {
	Type      string          `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Timezone  string          `json:"timezone,omitempty"`
	Year      int             `json:"year,omitempty"`
	Planets   []Position      `json:"planets,omitempty"`
	Houses    []House         `json:"houses,omitempty"`
	Aspects   []Aspect        `json:"aspects,omitempty"`
	Events    []CalendarEvent `json:"events,omitempty"`
	Slices    []TimeSlice     `json:"slices,omitempty"`
}

var planetNamesRu = map[string]string{
	"0": "Солнце", "Sun": "Солнце",
	"1": "Луна", "Moon": "Луна",
	"2": "Меркурий", "Mercury": "Меркурий",
	"3": "Венера", "Venus": "Венера",
	"4": "Марс", "Mars": "Марс",
	"5": "Юпитер", "Jupiter": "Юпитер",
	"6": "Сатурн", "Saturn": "Сатурн",
	"7": "Уран", "Uranus": "Уран",
	"8": "Нептун", "Neptune": "Нептун",
	"9": "Плутон", "Pluto": "Плутон",
	"10": "Раху", "Mean Node": "Раху",
	"11": "Кету", "True Node": "Кету",
	"12": "Лилит", "Lilith": "Лилит",
}

var aspectNamesRu = map[string]string{
	"0": "Соединение (0°)", "Conjunction": "Соединение (0°)",
	"60": "Секстиль (60°)", "Sextile": "Секстиль (60°)",
	"90": "Квадратура (90°)", "Square": "Квадратура (90°)",
	"120": "Трин (120°)", "Trine": "Трин (120°)",
	"180": "Оппозиция (180°)", "Opposition": "Оппозиция (180°)",
}

var zodiacSignsRu = []string{
	"Овен", "Телец", "Близнецы", "Рак", "Лев", "Дева",
	"Весы", "Скорпион", "Стрелец", "Козерог", "Водолей", "Рыбы",
}

func getPlanetRu(p string) string {
	if ru, ok := planetNamesRu[p]; ok {
		return ru
	}
	return p
}

func getAspectRu(a string) string {
	if ru, ok := aspectNamesRu[a]; ok {
		return ru
	}
	return a
}

func getSignRu(idx int) string {
	if idx >= 0 && idx < 12 {
		return zodiacSignsRu[idx]
	}
	return fmt.Sprintf("Знак %d", idx)
}

func (e CalendarEvent) String() string {
	parsedTime, err := time.Parse("2006-01-02T15:04", e.Date)
	timeStr := e.Date
	if err == nil {
		timeStr = parsedTime.Format("02.01.2006 15:04")
	}

	switch e.Type {
	case "chS": // Смена знака (Ингрессия)
		planet := "Планета"
		if len(e.Planets) > 0 {
			planet = getPlanetRu(e.Planets[0])
		}
		signIdx := 0
		if e.Sign != nil {
			signIdx = *e.Sign // Разыменовываем указатель (получаем число 0, 1, 2...)
		}

		sign := getSignRu(signIdx)
		return fmt.Sprintf("[%s] Ингрессия: %s переходит в знак %s", timeStr, planet, sign)
	case "r": // Смена направления движения (Разворот)
		planet := "Планета"
		if len(e.Planets) > 0 {
			planet = getPlanetRu(e.Planets[0])
		}

		motionType := "директный"
		if e.Aspect == "R" {
			motionType = "ретроградный"
		}

		return fmt.Sprintf("[%s] Разворот: %s %s", timeStr, planet, motionType)
	case "exA": // Точный аспект
		p1 := "Планета 1"
		p2 := "Планета 2"
		if len(e.Planets) >= 2 {
			p1 = getPlanetRu(e.Planets[0])
			p2 = getPlanetRu(e.Planets[1])
		}
		aspect := getAspectRu(e.Aspect)
		return fmt.Sprintf("[%s] Аспект: %s - %s (%s)", timeStr, p1, p2, aspect)

	case "lunD": // Лунный день
		return fmt.Sprintf("[%s] %d-й лунный день", timeStr, e.Sign)

	case "noC": // Луна без курса
		changeTimeStr := e.ChangeSign
		parsedChange, err := time.Parse("2006-01-02T15:04", e.ChangeSign)
		if err == nil {
			changeTimeStr = parsedChange.Format("15:04")
		} else if len(changeTimeStr) > 16 {
			changeTimeStr = changeTimeStr[11:16]
		}
		return fmt.Sprintf("[%s] Луна без курса (холостая) до %s", timeStr, changeTimeStr)

	default:
		return fmt.Sprintf("[%s] Событие %s: %v", timeStr, e.Type, e.Planets)
	}
}

// FilterConfig определяет параметры фильтрации для результатов расчетов
type FilterConfig struct {
	MaxOrb      *float64        // Фильтр по точности орбиса (например, 1.0). Если nil — фильтрация по орбису отключена
	Planets     map[string]bool // Разрешенные имена или ID планет (например, {"Sun": true, "Moon": true, "0": true, "1": true})
	AspectTypes map[string]bool // Разрешенные типы аспектов (например, {"Conjunction": true, "Trine": true})
	EventTypes  map[string]bool // Разрешенные типы событий календаря (например, {"lunD": true, "noC": true})
}

// ApplyFilter фильтрует данные в AstroResult в соответствии с FilterConfig
func (r *AstroResult) ApplyFilter(cfg FilterConfig) {
	if r == nil {
		return
	}

	// 1. Фильтрация планет (для Natal, Synastry, Period)
	if len(cfg.Planets) > 0 {
		if len(r.Planets) > 0 {
			var filteredPlanets []Position
			for _, p := range r.Planets {
				idStr := fmt.Sprintf("%d", p.ID)
				if cfg.Planets[p.Name] || cfg.Planets[idStr] {
					filteredPlanets = append(filteredPlanets, p)
				}
			}
			r.Planets = filteredPlanets
		}

		if len(r.Slices) > 0 {
			for idx := range r.Slices {
				var filteredPlanets []Position
				for _, p := range r.Slices[idx].Planets {
					idStr := fmt.Sprintf("%d", p.ID)
					if cfg.Planets[p.Name] || cfg.Planets[idStr] {
						filteredPlanets = append(filteredPlanets, p)
					}
				}
				r.Slices[idx].Planets = filteredPlanets
			}
		}
	}

	// Функция для проверки аспекта по правилам конфигурации
	shouldKeepAspect := func(a Aspect) bool {
		// Фильтр по типу аспекта
		if len(cfg.AspectTypes) > 0 {
			if !cfg.AspectTypes[a.Type] {
				return false
			}
		}
		// Фильтр по максимальному орбису
		if cfg.MaxOrb != nil {
			if a.Orb > *cfg.MaxOrb {
				return false
			}
		}
		// Фильтр по планетам, участвующим в аспекте
		if len(cfg.Planets) > 0 {
			p1Clean := strings.TrimSuffix(strings.TrimSuffix(a.Planet1, " (Transit)"), " (Natal)")
			p2Clean := strings.TrimSuffix(strings.TrimSuffix(a.Planet2, " (Transit)"), " (Natal)")
			p1IDStr := fmt.Sprintf("%d", a.Planet1ID)
			p2IDStr := fmt.Sprintf("%d", a.Planet2ID)

			p1Allowed := cfg.Planets[p1Clean] || cfg.Planets[p1IDStr]
			p2Allowed := cfg.Planets[p2Clean] || cfg.Planets[p2IDStr]

			if !p1Allowed || !p2Allowed {
				return false
			}
		}
		return true
	}

	// Применяем фильтрацию аспектов
	if len(r.Aspects) > 0 {
		var filteredAspects []Aspect
		for _, a := range r.Aspects {
			if shouldKeepAspect(a) {
				filteredAspects = append(filteredAspects, a)
			}
		}
		r.Aspects = filteredAspects
	}

	if len(r.Slices) > 0 {
		for idx := range r.Slices {
			if len(r.Slices[idx].Aspects) > 0 {
				var filteredAspects []Aspect
				for _, a := range r.Slices[idx].Aspects {
					if shouldKeepAspect(a) {
						filteredAspects = append(filteredAspects, a)
					}
				}
				r.Slices[idx].Aspects = filteredAspects
			}
		}
	}

	// 2. Фильтрация событий календаря
	if len(r.Events) > 0 {
		var filteredEvents []CalendarEvent
		for _, e := range r.Events {
			// Фильтр по типу события
			if len(cfg.EventTypes) > 0 {
				if !cfg.EventTypes[e.Type] {
					continue
				}
			}
			// Фильтр по планетам в событии
			if len(cfg.Planets) > 0 {
				if len(e.Planets) > 0 {
					keepEvent := false
					for _, ep := range e.Planets {
						epRu := getPlanetRu(ep)
						idStr := ep
						if id, err := strconv.Atoi(ep); err == nil {
							idStr = GetPlanetName(id)
						}
						if cfg.Planets[ep] || cfg.Planets[epRu] || cfg.Planets[idStr] {
							keepEvent = true
							break
						}
					}
					if !keepEvent {
						continue
					}
				} else if e.Type == "lunD" {
					// Лунные дни привязаны к Луне ("1" или "Moon")
					if !cfg.Planets["1"] && !cfg.Planets["Moon"] && !cfg.Planets["Луна"] {
						continue
					}
				}
			}
			filteredEvents = append(filteredEvents, e)
		}
		r.Events = filteredEvents
	}
}
