package output

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"math"
	"os/exec"

	"github.com/damirmur/swisseph_build/pkg/astro"
)

type ImageRenderer struct {
	ConvertToPNG bool
}

// chartTemplate начинается строго соsvg и гарантированно закрывает все теги
const chartTemplate = `<svg width="600" height="600" viewBox="0 0 600 600" xmlns="http://www.w3.org/2000/svg" style="background-color: #1a202c;">
	<!-- Центральная точка холста -->
	<circle cx="300" cy="300" r="240" stroke="#4a5568" stroke-width="2" fill="none" />
	<circle cx="300" cy="300" r="180" stroke="#4a5568" stroke-width="1" fill="none" />
	<circle cx="300" cy="300" r="120" stroke="#4a5568" stroke-width="1" fill="none" />

	<!-- 12 секторов Знаков Зодиака -->
	{{range $i := seq 0 11}}
		{{drawZodiacLine $i}}
	{{end}}

	<!-- Отрисовка Домов -->
	{{range .Houses}}
		{{drawHouseLine .Number .Longitude}}
	{{end}}

	<!-- Отрисовка Аспектов -->
	{{range .Aspects}}
		{{drawAspectLine .Planet1 .Planet2 .Type}}
	{{end}}

	<!-- Отрисовка Планет -->
	{{range .Planets}}
		{{drawPlanet .Name .Longitude}}
	{{end}}
</svg>`

func (r *ImageRenderer) Render(ctx context.Context, result *astro.AstroResult, w io.Writer) error {
	if result == nil {
		return fmt.Errorf("данные для построения карты отсутствуют")
	}

	// Записываем XML-декларацию напрямую
	_, err := w.Write([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"no\"?>\n"))
	if err != nil {
		return fmt.Errorf("ошибка записи XML заголовка: %w", err)
	}

	planetCoords := make(map[string]float64)
	for _, p := range result.Planets {
		planetCoords[p.Name] = p.Longitude
	}

	funcMap := template.FuncMap{
		"seq": func(start, end int) []int {
			var res []int
			for i := start; i <= end; i++ {
				res = append(res, i)
			}
			return res
		},
		"drawZodiacLine": func(idx int) template.HTML {
			angle := float64(idx * 30)
			x1, y1 := getCirclePoint(300, 300, 180, angle)
			x2, y2 := getCirclePoint(300, 300, 240, angle)
			return template.HTML(fmt.Sprintf(`<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="#4a5568" stroke-width="1" stroke-dasharray="2,2" />`, x1, y1, x2, y2))
		},
		"drawHouseLine": func(num int, lon float64) template.HTML {
			x1, y1 := getCirclePoint(300, 300, 120, lon)
			x2, y2 := getCirclePoint(300, 300, 240, lon)
			color := "#718096"
			width := 1
			if num == 1 || num == 10 {
				color = "#ecc94b"
				width = 2
			}
			return template.HTML(fmt.Sprintf(`<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="%s" stroke-width="%d" />`, x1, y1, x2, y2, color, width))
		},
		"drawPlanet": func(name string, lon float64) template.HTML {
			// Обычные координаты маркера планеты (радиус 210)
			x, y := getCirclePoint(300, 300, 210, lon)

			// Вычисляем динамический радиус для текста на основе названия планеты,
			// чтобы избежать наложения (например, Солнце и Венера разойдутся по высоте)
			textRadius := 228.0
			if name == "Venus" || name == "Mercury" || name == "Neptune" {
				textRadius = 242.0 // Сдвигаем чуть дальше наружу
			}

			xt, yt := getCirclePoint(300, 300, textRadius, lon)

			displayName := name
			if len(name) > 2 {
				displayName = name[:2]
			}

			rawHTML := fmt.Sprintf(`<circle cx="%.2f" cy="%.2f" r="4" fill="#63b3ed" /><text x="%.2f" y="%.2f" fill="#fff" font-size="11" font-family="sans-serif" text-anchor="middle" font-weight="bold">%s</text>`,
				x, y, xt, yt, displayName)
			return template.HTML(rawHTML)
		},
		"drawAspectLine": func(p1Name, p2Name, aspType string) template.HTML {
			// Очищаем имена планет от суффиксов транзита/натала для поиска координат
			cleanName := func(name string) string {
				// Если имя содержит пробел (например, "Sun (Transit)"), берем только первое слово
				for i, char := range name {
					if char == ' ' {
						return name[:i]
					}
				}
				return name
			}

			n1 := cleanName(p1Name)
			n2 := cleanName(p2Name)

			lon1, ok1 := planetCoords[n1]
			lon2, ok2 := planetCoords[n2]
			if !ok1 || !ok2 {
				return template.HTML("") // Если планета не найдена, линию не рисуем
			}

			// Линии аспектов рисуются внутри центрального круга (радиус 120)
			x1, y1 := getCirclePoint(300, 300, 120, lon1)
			x2, y2 := getCirclePoint(300, 300, 120, lon2)

			// Цветовое кодирование мажорных аспектов
			color := "#48bb78" // Зеленый для гармоничных (Trine, Sextile)
			if aspType == "Square" || aspType == "Opposition" {
				color = "#f56565" // Красный для напряженных
			} else if aspType == "Conjunction" {
				color = "#ed64a6" // Розовый для соединений
			}

			rawHTML := fmt.Sprintf(`<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="%s" stroke-width="1.5" opacity="0.75" />`, x1, y1, x2, y2, color)
			return template.HTML(rawHTML)
		}}

	tmpl, err := template.New("svg_chart").Funcs(funcMap).Parse(chartTemplate)
	if err != nil {
		return fmt.Errorf("ошибка парсинга SVG шаблона: %w", err)
	}

	return tmpl.Execute(w, result)
}

func getCirclePoint(cx, cy, r, angleData float64) (float64, float64) {
	rad := (angleData - 180) * math.Pi / 180.0
	x := cx + r*math.Cos(rad)
	y := cy + r*math.Sin(rad)
	return x, y
}

func ConvertSvgToPng(svgPath, pngPath string) error {
	cmd := exec.Command("resvg", svgPath, pngPath)
	if err := cmd.Run(); err != nil {
		cmdFallback := exec.Command("rsvg-convert", "-o", pngPath, svgPath)
		return cmdFallback.Run()
	}
	return nil
}
