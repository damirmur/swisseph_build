// pkg/astro/calendar_response.go
package astro

// CompactEvent — компактное событие календаря в форме, готовой к отдаче
// клиенту (без ремаппинга на стороне фронта). Ключи совпадают с полями meta.fields.
type CompactEvent struct {
	Date       string   `json:"d"`
	Type       string   `json:"t"`
	Planets    []string `json:"p,omitempty"`
	Aspect     string   `json:"a,omitempty"`
	Degrees    []int    `json:"de,omitempty"`
	Sign       *int     `json:"s,omitempty"`
	ChangeSign string   `json:"chS,omitempty"`
	Retro      *int     `json:"r,omitempty"`
	Houses     []int    `json:"h,omitempty"` // полнознаковые дома (совмещены с p), только при заданном sign
}

// CalendarResponse — самоописываемая огибающая ответа /api/v1/calendar.
// Исключает ремаппинг на клиенте: data содержит готовые события,
// meta — словари и описание структуры.
type CalendarResponse struct {
	Schema  string         `json:"schema"`
	Meta    Meta           `json:"meta"`
	AscSign *int           `json:"asc_sign,omitempty"` // знак Асцендента (0-11), если запрошены дома
	Data    []CompactEvent `json:"data"`
}

const CalendarSchemaVersion = "astro3d.calendar/v1"

// WholeSignHouse возвращает номер полнознакового дома (1-12) для планеты в знаке
// planetSign относительно Асцендента в знаке ascSign (оба — 0-индексные знаки Зодиака).
func WholeSignHouse(planetSign, ascSign int) int {
	return ((planetSign-ascSign)%12 + 12) % 12 + 1
}

// planetSigns извлекает знаки Зодиака участников события, совмещённые по порядку с e.Planets.
// Для exA/chS/r знаки берутся из долгот (Degrees); для lunD/noC планет нет — возвращается nil.
func planetSigns(e CalendarEvent) []int {
	if len(e.Planets) == 0 {
		return nil
	}
	out := make([]int, len(e.Planets))
	for k := range e.Planets {
		sign := -1
		switch {
		case len(e.Degrees) > k:
			sign = e.Degrees[k] / 30
		case e.Sign != nil && k == 0:
			sign = *e.Sign
		}
		out[k] = sign
	}
	return out
}

// toCompact базовое преобразование без домов.
func toCompact(e CalendarEvent) CompactEvent {
	ce := CompactEvent{
		Date:       e.Date,
		Type:       e.Type,
		Planets:    e.Planets,
		Aspect:     e.Aspect,
		Degrees:    e.Degrees,
		Sign:       e.Sign,
		ChangeSign: e.ChangeSign,
	}
	if e.Type == "r" {
		val := 0
		if e.Aspect == "R" {
			val = 1
		}
		ce.Retro = &val
		ce.Aspect = ""
	}
	return ce
}

// ToCompactEvents преобразует события календаря в компактную форму.
func ToCompactEvents(events []CalendarEvent) []CompactEvent {
	out := make([]CompactEvent, 0, len(events))
	for _, e := range events {
		out = append(out, toCompact(e))
	}
	return out
}

// ToCompactEventsForSign — как ToCompactEvents, но дополняет события exA/chS/r
// полнознаковыми домами (поле h) относительно Асцендента в знаке ascSign.
func ToCompactEventsForSign(events []CalendarEvent, ascSign int) []CompactEvent {
	out := make([]CompactEvent, 0, len(events))
	for _, e := range events {
		ce := toCompact(e)
		if (e.Type == "exA" || e.Type == "chS" || e.Type == "r") && len(e.Planets) > 0 {
			signs := planetSigns(e)
			houses := make([]int, 0, len(signs))
			for _, s := range signs {
				if s < 0 {
					houses = append(houses, 0)
				} else {
					houses = append(houses, WholeSignHouse(s, ascSign))
				}
			}
			ce.Houses = houses
		}
		out = append(out, ce)
	}
	return out
}

// NewCalendarResponse строит самоописываемый ответ календаря.
func NewCalendarResponse(events []CalendarEvent) CalendarResponse {
	return CalendarResponse{
		Schema: CalendarSchemaVersion,
		Meta:   NewCalendarMeta(),
		Data:   ToCompactEvents(events),
	}
}

// NewCalendarResponseForSign строит ответ с полнознаковыми домами относительно ascSign.
func NewCalendarResponseForSign(events []CalendarEvent, ascSign int) CalendarResponse {
	asc := ascSign
	return CalendarResponse{
		Schema:  CalendarSchemaVersion,
		Meta:    NewCalendarMeta(),
		AscSign: &asc,
		Data:    ToCompactEventsForSign(events, ascSign),
	}
}
