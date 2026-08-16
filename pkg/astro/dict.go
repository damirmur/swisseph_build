// pkg/astro/dict.go
package astro

// Canonical dictionaries — единственный источник истины для кодов API.
// Используются всеми эндпоинтами (natal/calendar/period) и фронтендом
// через поле meta ответа, устраняя дублирование справочников в клиенте.

// PlanetDict — код планеты -> английское имя.
var PlanetDict = map[string]string{
	"0":  "Sun",
	"1":  "Moon",
	"2":  "Mercury",
	"3":  "Venus",
	"4":  "Mars",
	"5":  "Jupiter",
	"6":  "Saturn",
	"7":  "Uranus",
	"8":  "Neptune",
	"9":  "Pluto",
	"10": "Mean Node",
	"11": "North Node",
	"12": "Lilith",
	"13": "Osc. Apogee",
}

// SignDict — код знака Зодиака -> английское имя.
var SignDict = map[string]string{
	"0":  "Aries",
	"1":  "Taurus",
	"2":  "Gemini",
	"3":  "Cancer",
	"4":  "Leo",
	"5":  "Virgo",
	"6":  "Libra",
	"7":  "Scorpio",
	"8":  "Sagittarius",
	"9":  "Capricorn",
	"10": "Aquarius",
	"11": "Pisces",
}

// AspectDict — угол аспекта (градусы) -> английское имя.
var AspectDict = map[string]string{
	"0":   "Conjunction",
	"30":  "Semi-sextile",
	"60":  "Sextile",
	"72":  "Quintile",
	"90":  "Square",
	"120": "Trine",
	"135": "Sesquiquadrate",
	"180": "Opposition",
}

// EventTypeDict — тип события календаря -> описание.
var EventTypeDict = map[string]string{
	"exA": "Exact aspect between planets",
	"lunD": "Lunar day (Moon enters a new lunar day)",
	"chS": "Planet changes zodiac sign (ingress)",
	"noC": "Moon void of course",
	"r":   "Direction change (direct <-> retrograde)",
}

// RetroDict — значение поля r для событий типа "r".
var RetroDict = map[string]string{
	"0": "Direct",
	"1": "Retrograde",
}

// MetaField описывает одно поле события календаря (самоописание структуры).
type MetaField struct {
	Type        string            `json:"type"`
	Format      string            `json:"format,omitempty"`
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	Ref         string            `json:"ref,omitempty"`   // ссылка на словарь в meta для расшифровки кодов
	Enum        map[string]string `json:"enum,omitempty"`  // прямая расшифровка значений
	Optional    bool              `json:"optional,omitempty"`
	Example     string            `json:"example,omitempty"`
}

// CalendarMeta — блок meta самоописываемого ответа календаря.
type CalendarMeta struct {
	Planets map[string]string   `json:"planets"`
	Signs   map[string]string   `json:"signs"`
	Aspects map[string]string   `json:"aspects"`
	Types   map[string]string   `json:"types"`
	Fields  map[string]MetaField `json:"fields"`
	Params  map[string]string   `json:"params,omitempty"` // описание query-параметров API
}

// NewCalendarMeta возвращает заполненный блок meta для ответа календаря.
func NewCalendarMeta() CalendarMeta {
	return CalendarMeta{
		Planets: PlanetDict,
		Signs:   SignDict,
		Aspects: AspectDict,
		Types:   EventTypeDict,
		Fields: map[string]MetaField{
			"d": {
				Type:     "string",
				Format:   "date-time",
				Title:    "Дата/время события (ISO-8601, UTC)",
				Example:  "2026-01-01T13:33:00Z",
			},
			"t": {
				Type:  "string",
				Title: "Тип события",
				Ref:   "types",
			},
			"p": {
				Type:        "array",
				Title:       "Планеты (коды)",
				Description: "Массив кодов планет; null/отсутствует, если событие не привязано к планетам",
				Ref:         "planets",
				Optional:    true,
			},
			"a": {
				Type:     "string",
				Title:    "Угол аспекта (градусы)",
				Ref:      "aspects",
				Optional: true,
			},
			"de": {
				Type:        "array",
				Title:       "Долготы участников (градусы)",
				Description: "Долготы планет/куспидов, участвующих в событии",
				Optional:    true,
			},
			"s": {
				Type:     "integer",
				Title:    "Знак Зодиака",
				Ref:      "signs",
				Optional: true,
			},
			"chS": {
				Type:     "string",
				Format:   "date-time",
				Title:    "Время смены знака (для типа noC)",
				Optional: true,
			},
			"r": {
				Type:        "integer",
				Title:       "Направление движения (для типа r): 0 = прямое, 1 = ретроградное",
				Enum:        RetroDict,
				Optional:    true,
			},
			"h": {
				Type:        "array",
				Title:       "Полнознаковые дома (номера 1-12), совмещены по порядку с полем p",
				Description: "Заполняется только если в запросе задан параметр sign (знак Асцендента).",
				Optional:    true,
			},
		},
		Params: map[string]string{
			"year":        "Год (используется, если не заданы start/end)",
			"month":       "Месяц 1-12 (0 или пусто — весь год)",
			"start":       "Начало произвольного периода (ISO-8601), интерпретируется в часовом поясе city",
			"end":         "Конец произвольного периода (ISO-8601)",
			"sign":        "Знак Асцендента 0-11 (Aries..Pisces) — добавляет полнознаковые дома в поле h",
			"city":        "Город для определения часового пояса (по умолчанию UTC)",
			"event_types": "Фильтр типов событий: exA,lunD,chS,noC,r",
			"planets":     "Фильтр планет (коды через запятую)",
			"aspects":     "Фильтр аспектов (углы через запятую)",
		},
	}
}
