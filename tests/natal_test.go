package tests

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/damirmur/swisseph_build/pkg/astro"
	"github.com/damirmur/swisseph_build/pkg/output"
)

// natalReferenceText — эталонный вывод `./astro natal --date "1971-11-19 19:43" --lat "51.77" --lon "55.10" --format text`.
// Сгенерирован работающей реализацией (система домов по умолчанию "P" — Placidus).
const natalReferenceText = `==================================================================
АНАЛИТИЧЕСКИЙ ОТЧЕТ: НАТАЛЬНАЯ КАРТА
Дата расчета (UTC): 1971-11-19 19:43:00
==================================================================

[ПОЛОЖЕНИЕ НЕБЕСНЫХ ТЕЛ В ЗНАКАХ И ДОМАХ]
• Sun        находится в 26° Скорпиона (Долгота: 236.821°, Дом: 4)
• Moon       находится в 16° Стрельца (Долгота: 256.528°, Дом: 4)
• Mercury    находится в 18° Стрельца (Долгота: 258.289°, Дом: 4)
• Venus      находится в 18° Стрельца (Долгота: 258.436°, Дом: 4)
• Mars       находится в 07° Рыб      (Долгота: 337.325°, Дом: 7)
• Jupiter    находится в 12° Стрельца (Долгота: 252.866°, Дом: 4)
• Saturn     находится в 03° Близнецов (Долгота:  63.530°, Дом: 10)
• Uranus     находится в 16° Весов    (Долгота: 196.503°, Дом: 2)
• Neptune    находится в 02° Стрельца (Долгота: 242.578°, Дом: 4)
• Pluto      находится в 01° Весов    (Долгота: 181.466°, Дом: 2)
• Mean Node  находится в 08° Водолея  (Долгота: 308.866°, Дом: 6)
• Lilith     находится в 19° Весов    (Долгота: 199.204°, Дом: 3)

[ГЕОМЕТРИЧЕСКИЕ СВЯЗИ И АСПЕКТЫ]
  - Взаимодействие Sun и Neptune через аспект Соединение (0°) (Точность/Орбис: 5.76°)
  - Взаимодействие Moon и Mercury через аспект Соединение (0°) (Точность/Орбис: 1.76°)
  - Взаимодействие Moon и Venus через аспект Соединение (0°) (Точность/Орбис: 1.91°)
  - Взаимодействие Moon и Jupiter через аспект Соединение (0°) (Точность/Орбис: 3.66°)
  - Взаимодействие Moon и Uranus через аспект Секстиль (60°) (Точность/Орбис: 0.02°)
  - Взаимодействие Moon и Lilith через аспект Секстиль (60°) (Точность/Орбис: 2.68°)
  - Взаимодействие Mercury и Venus через аспект Соединение (0°) (Точность/Орбис: 0.15°)
  - Взаимодействие Mercury и Jupiter через аспект Соединение (0°) (Точность/Орбис: 5.42°)
  - Взаимодействие Mercury и Uranus через аспект Секстиль (60°) (Точность/Орбис: 1.79°)
  - Взаимодействие Mercury и Lilith через аспект Секстиль (60°) (Точность/Орбис: 0.91°)
  - Взаимодействие Venus и Jupiter через аспект Соединение (0°) (Точность/Орбис: 5.57°)
  - Взаимодействие Venus и Uranus через аспект Секстиль (60°) (Точность/Орбис: 1.93°)
  - Взаимодействие Venus и Lilith через аспект Секстиль (60°) (Точность/Орбис: 0.77°)
  - Взаимодействие Mars и Saturn через аспект Квадратура (90°) (Точность/Орбис: 3.79°)
  - Взаимодействие Mars и Neptune через аспект Квадратура (90°) (Точность/Орбис: 4.75°)
  - Взаимодействие Jupiter и Uranus через аспект Секстиль (60°) (Точность/Орбис: 3.64°)
  - Взаимодействие Jupiter и Mean Node через аспект Секстиль (60°) (Точность/Орбис: 4.00°)
  - Взаимодействие Saturn и Neptune через аспект Оппозиция (180°) (Точность/Орбис: 0.95°)
  - Взаимодействие Saturn и Pluto через аспект Трин (120°) (Точность/Орбис: 2.06°)
  - Взаимодействие Saturn и Mean Node через аспект Трин (120°) (Точность/Орбис: 5.34°)
  - Взаимодействие Uranus и Lilith через аспект Соединение (0°) (Точность/Орбис: 2.70°)
  - Взаимодействие Neptune и Pluto через аспект Секстиль (60°) (Точность/Орбис: 1.11°)
`

// computeNatalText воспроизводит расчёт натальной карты 19.11.1971 19:43 UTC
// (51.77°N, 55.10°E, дома Placidus) и рендерит её в текстовый формат.
func computeNatalText(t *testing.T) string {
	t.Helper()
	ephePath, err := filepath.Abs("../ephe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ephePath); err != nil {
		t.Skipf("каталог эфемерид %q не найден — тест пропущен", ephePath)
	}

	tm := time.Date(1971, 11, 19, 19, 43, 0, 0, time.UTC)

	calc := astro.NewCalculator(ephePath)
	defer calc.Close()
	res, err := calc.ComputeNatal(context.Background(), tm, 51.77, 55.10, "P")
	if err != nil {
		t.Fatalf("ComputeNatal: %v", err)
	}

	render, err := output.GetRenderer("text")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := render.Render(context.Background(), res, &buf); err != nil {
		t.Fatalf("Render: %v", err)
	}
	return buf.String()
}

func TestNatalPlanetAndAspectCounts(t *testing.T) {
	ephePath, err := filepath.Abs("../ephe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ephePath); err != nil {
		t.Skipf("каталог эфемерид %q не найден — тест пропущен", ephePath)
	}

	tm := time.Date(1971, 11, 19, 19, 43, 0, 0, time.UTC)
	calc := astro.NewCalculator(ephePath)
	defer calc.Close()
	res, err := calc.ComputeNatal(context.Background(), tm, 51.77, 55.10, "P")
	if err != nil {
		t.Fatalf("ComputeNatal: %v", err)
	}

	if got := len(res.Planets); got != 12 {
		t.Errorf("количество планет не совпадает: эталон 12, получено %d", got)
	}
	if got := len(res.Aspects); got != 22 {
		t.Errorf("количество аспектов не совпадает: эталон 22, получено %d", got)
	}
	t.Logf("планет: %d, аспектов: %d — совпадает", len(res.Planets), len(res.Aspects))
}

func TestNatalMatchesReference(t *testing.T) {
	actual := computeNatalText(t)
	expected := natalReferenceText

	if actual == expected {
		t.Log("натальная карта полностью совпадает с эталоном")
		return
	}

	expLines := strings.Split(expected, "\n")
	actLines := strings.Split(actual, "\n")
	max := len(expLines)
	if len(actLines) > max {
		max = len(actLines)
	}

	mismatches := 0
	for i := 0; i < max; i++ {
		hasExp := i < len(expLines)
		hasAct := i < len(actLines)
		exp, act := "", ""
		if hasExp {
			exp = expLines[i]
		}
		if hasAct {
			act = actLines[i]
		}
		if exp == act {
			continue
		}
		mismatches++
		if mismatches > 30 {
			continue
		}
		switch {
		case hasExp && hasAct:
			t.Errorf("строка %d:\n  эталон:  %s\n  текущее: %s", i+1, exp, act)
		case hasExp:
			t.Errorf("строка %d отсутствует в текущем выводе:\n  эталон: %s", i+1, exp)
		default:
			t.Errorf("лишняя строка %d:\n  текущее: %s", i+1, act)
		}
	}
	t.Errorf("всего расхождений: %d", mismatches)
}
