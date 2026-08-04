package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/damirmur/swisseph_build/pkg/astro"
	"github.com/damirmur/swisseph_build/pkg/interactive"
	"github.com/damirmur/swisseph_build/pkg/output"
	"github.com/damirmur/swisseph_build/pkg/storage"

	"github.com/spf13/cobra"
)

var (
	outputFormat string
	saveDir      string
	hsys         string
	lat, lon     float64
	dateStr      string
	dateStr2     string
	periodStart  string
	periodEnd    string
	stepHours    int
	calYear      int
	calMonth     int
	calCity      string
)

// GetExecutableDir возвращает абсолютный путь к папке, где лежит исполняемый файл
func GetExecutableDir() string {
	exePath, err := os.Executable()
	if err != nil {
		log.Printf("Ошибка получения пути процесса: %v, используем текущую директорию", err)
		return "."
	}
	return filepath.Dir(exePath)
}

// epheDir возвращает путь к каталогу эфемерид рядом с исполняемым файлом
func epheDir() string {
	return filepath.Join(GetExecutableDir(), "ephe")
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	exeDir := GetExecutableDir()
	defaultSaveDir := filepath.Join(exeDir, "output_data")

	var rootCmd = &cobra.Command{
		Use:   "astro",
		Short: "Astro - астрологический вычислитель",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	rootCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", "console", "Формат: console, json, text, image")
	rootCmd.PersistentFlags().StringVarP(&saveDir, "out-dir", "o", defaultSaveDir, "Папка для сохранения файлов на VPS")
	rootCmd.PersistentFlags().StringVar(&hsys, "hsys", "P", "Система домов (P-Placidus, K-Koch, etc.)")

	// Настройка хранилища после парсинга флагов (cobra парсит флаги при rootCmd.Execute)
	// Для обратной совместимости создадим хранилище и запустим очистку прямо перед выполнением команд
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		store := storage.New(saveDir, 7)
		store.StartAutoCleanup(ctx, 12*time.Hour)
	}

	var natalCmd = &cobra.Command{
		Use:   "natal",
		Short: "Вычислить натальную карту",
		Run: func(cmd *cobra.Command, args []string) {
			if dateStr == "" {
				runInteractiveNatal()
			} else {
				executeAstroJob("natal")
			}
		},
	}
	natalCmd.Flags().StringVar(&dateStr, "date", "", "Дата и время в UTC (YYYY-MM-DD HH:MM)")
	natalCmd.Flags().Float64Var(&lat, "lat", 55.75, "Широта")
	natalCmd.Flags().Float64Var(&lon, "lon", 37.61, "Долгота")

	var synastryCmd = &cobra.Command{
		Use:   "synastry",
		Short: "Вычислить синастрию или транзит двух карт",
		Run: func(cmd *cobra.Command, args []string) {
			if dateStr == "" || dateStr2 == "" {
				runInteractiveSynastry()
			} else {
				executeAstroJob("synastry")
			}
		},
	}
	synastryCmd.Flags().StringVar(&dateStr, "date1", "", "Дата Натала в UTC (YYYY-MM-DD HH:MM)")
	synastryCmd.Flags().StringVar(&dateStr2, "date2", "", "Дата Транзита/Партнера в UTC (YYYY-MM-DD HH:MM)")
	synastryCmd.Flags().Float64Var(&lat, "lat", 55.75, "Широта Натала")
	synastryCmd.Flags().Float64Var(&lon, "lon", 37.61, "Долгота Натала")

	var periodCmd = &cobra.Command{
		Use:   "period",
		Short: "Положение планет и аспектов за диапазон дат",
		Run: func(cmd *cobra.Command, args []string) {
			if periodStart == "" || periodEnd == "" {
				fmt.Println("Не указаны даты для расчета периода.")
				cmd.Help()
			} else {
				executeAstroJob("period")
			}
		},
	}
	periodCmd.Flags().StringVar(&periodStart, "start", "", "Старт периода UTC (YYYY-MM-DD HH:MM)")
	periodCmd.Flags().StringVar(&periodEnd, "end", "", "Конец периода UTC (YYYY-MM-DD HH:MM)")
	periodCmd.Flags().IntVar(&stepHours, "step", 24, "Шаг расчета в часах (например: 1, 6, 24)")

	var calendarCmd = &cobra.Command{
		Use:   "calendar",
		Short: "Календарь астрособытий за месяц или год",
		Run: func(cmd *cobra.Command, args []string) {
			if !cmd.Flags().Changed("year") && !cmd.Flags().Changed("month") {
				runInteractiveCalendar()
			} else {
				executeAstroJob("calendar")
			}
		},
	}
	calendarCmd.Flags().IntVar(&calYear, "year", time.Now().Year(), "Год расчета календаря")
	calendarCmd.Flags().IntVar(&calMonth, "month", 0, "Месяц расчета (1-12). Если 0 — считает за весь год")
	calendarCmd.Flags().StringVar(&calCity, "city", "", "Город для расчета времени событий (по умолчанию UTC+0)")

	rootCmd.AddCommand(natalCmd, synastryCmd, periodCmd, calendarCmd)
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runInteractiveNatal() {
	fmt.Println("\n=== Натал ===")
	input, _ := interactive.CollectData(true)
	gd, err := interactive.GetGeoData(input)
	if err != nil {
		log.Fatalf("Ошибка получения геоданных: %v", err)
	}

	if interactive.ConfirmData(input, gd) {
		t, _ := time.Parse("2006-01-02 15:04", input.Date+" "+input.Time)
		if input.IsLocal {
			t = t.Add(time.Duration(-gd.UTCOffset) * time.Hour)
		}

		calc := astro.NewCalculator(epheDir())
		defer calc.Close()
		result, err := calc.ComputeNatal(context.Background(), t, gd.Latitude, gd.Longitude, hsys)
		if err != nil {
			log.Fatalf("Ошибка расчета: %v", err)
		}

		renderConsole(result)
	}
}

func runInteractiveSynastry() {
	fmt.Println("\n=== Первая карта (Натал) ===")
	input1, _ := interactive.CollectData(false)
	gd1, err := interactive.GetGeoData(input1)
	if err != nil {
		log.Fatalf("Ошибка получения геоданных для карты 1: %v", err)
	}

	fmt.Println("\n=== Вторая карта (Транзит/Партнер) ===")
	input2, _ := interactive.CollectData(true)
	gd2, err := interactive.GetGeoData(input2)
	if err != nil {
		log.Fatalf("Ошибка получения геоданных для карты 2: %v", err)
	}

	fmt.Println("\nПодтверждение первой карты:")
	conf1 := interactive.ConfirmData(input1, gd1)
	fmt.Println("\nПодтверждение второй карты:")
	conf2 := interactive.ConfirmData(input2, gd2)

	if conf1 && conf2 {
		t1, _ := time.Parse("2006-01-02 15:04", input1.Date+" "+input1.Time)
		if input1.IsLocal {
			t1 = t1.Add(time.Duration(-gd1.UTCOffset) * time.Hour)
		}

		t2, _ := time.Parse("2006-01-02 15:04", input2.Date+" "+input2.Time)
		if input2.IsLocal {
			t2 = t2.Add(time.Duration(-gd2.UTCOffset) * time.Hour)
		}

		calc := astro.NewCalculator(epheDir())
		defer calc.Close()
		result, err := calc.ComputeSynastry(context.Background(), t1, t2, gd1.Latitude, gd1.Longitude, hsys)
		if err != nil {
			log.Fatalf("Ошибка расчета: %v", err)
		}

		renderConsole(result)
	}
}

func getLocationAndTZ(city string, tRef time.Time) (*time.Location, string, error) {
	if city == "" {
		return time.UTC, "UTC+0", nil
	}

	gd, err := interactive.GetGeoData(&interactive.UserInput{
		City: city,
		Date: tRef.Format("2006-01-02"),
		Time: tRef.Format("15:04"),
	})
	if err != nil {
		return nil, "", err
	}

	loc, err := time.LoadLocation(gd.Timezone)
	if err != nil {
		offsetSeconds := int(gd.UTCOffset * 3600)
		loc = time.FixedZone(gd.Timezone, offsetSeconds)
	}

	_, offset := tRef.In(loc).Zone()
	offsetHours := float64(offset) / 3600.0
	tzStr := "UTC+0"
	if offsetHours != 0 {
		tzStr = fmt.Sprintf("UTC%+g", offsetHours)
	}
	if gd.City != "" {
		tzStr = fmt.Sprintf("%s, %s", gd.City, tzStr)
	}

	return loc, tzStr, nil
}

func runInteractiveCalendar() {
	fmt.Println("\n=== Интерактивный Календарь ===")
	// Собираем данные через интерактивный ввод
	year, month, city := interactive.CollectCalendarData()

	fmt.Printf("\n--- Подтверждение данных ---\n")
	if month > 0 {
		fmt.Printf("Период: %02d.%d\n", month, year)
	} else {
		fmt.Printf("Период: весь %d год\n", year)
	}
	if city != "" {
		fmt.Printf("Город: %s\n", city)
	} else {
		fmt.Println("Город: не указан (UTC+0)")
	}

	fmt.Print("\nВерно? (y/n) [Enter - да]: ")
	var confirm string
	fmt.Scanln(&confirm)

	if confirm == "" || confirm == "y" || confirm == "yes" || confirm == "д" || confirm == "да" {
		// Записываем собранные интерактивно данные в глобальные переменные флагов
		calYear = year
		calMonth = month
		calCity = city

		// Передаем управление в единую точку выполнения расчетов
		executeAstroJob("calendar")
	} else {
		fmt.Println("Операция отменена пользователем.")
	}
}

func executeAstroJob(jobType string) {
	ctx := context.Background()
	calc := astro.NewCalculator(epheDir())
	defer calc.Close()

	var result *astro.AstroResult
	var err error

	switch jobType {
	case "natal":
		t, _ := time.Parse("2006-01-02 15:04", dateStr)
		result, err = calc.ComputeNatal(ctx, t, lat, lon, hsys)
	case "synastry":
		t1, _ := time.Parse("2006-01-02 15:04", dateStr)
		t2, _ := time.Parse("2006-01-02 15:04", dateStr2)
		result, err = calc.ComputeSynastry(ctx, t1, t2, lat, lon, hsys)
	case "period":
		tStart, _ := time.Parse("2006-01-02 15:04", periodStart)
		tEnd, _ := time.Parse("2006-01-02 15:04", periodEnd)
		result, err = calc.ComputePeriod(ctx, tStart, tEnd, stepHours)
	case "calendar":
		var tStart, tEnd time.Time
		if calMonth > 0 {
			tStart = time.Date(calYear, time.Month(calMonth), 1, 0, 0, 0, 0, time.UTC)
			tEnd = tStart.AddDate(0, 1, 0)
		} else {
			tStart = time.Date(calYear, 1, 1, 0, 0, 0, 0, time.UTC)
			tEnd = time.Date(calYear, 12, 31, 23, 59, 0, 0, time.UTC)
		}

		loc, tzStr, err := getLocationAndTZ(calCity, tStart)
		if err != nil {
			log.Fatalf("Ошибка определения геоданных: %v", err)
		}

		tStart = tStart.Add(-24 * time.Hour)
		tEnd = tEnd.Add(24 * time.Hour)
		result, err = calc.ComputeCalendar(ctx, tStart, tEnd, loc)
		if err == nil && result != nil {
			result.Timezone = tzStr
			result.Year = calYear
		}
	}

	if err != nil {
		log.Fatalf("Ошибка расчета: %v", err)
	}

	saveResult(ctx, result)
}

// renderConsole выводит результат в консоль (используется интерактивными потоками).
func renderConsole(result *astro.AstroResult) {
	render, _ := output.GetRenderer("console")
	_ = render.Render(context.Background(), result, os.Stdout)
}

// saveResult выводит результат в консоль или сохраняет его в файл в зависимости от формата.
func saveResult(ctx context.Context, result *astro.AstroResult) {
	render, err := output.GetRenderer(outputFormat)
	if err != nil {
		log.Fatal(err)
	}

	if outputFormat == "console" {
		render.Render(ctx, result, os.Stdout)
		return
	}

	_ = os.MkdirAll(saveDir, 0755)

	if outputFormat == "svg" || outputFormat == "png" {
		svgFileName := fmt.Sprintf("%s_%d.svg", result.Type, time.Now().Unix())
		svgPath := filepath.Join(saveDir, svgFileName)

		f, _ := os.Create(svgPath)
		render.Render(ctx, result, f)
		f.Close()

		if outputFormat == "png" {
			pngPath := svgPath[:len(svgPath)-4] + ".png"
			if err := output.ConvertSvgToPng(svgPath, pngPath); err != nil {
				exec.Command("./resvg", svgPath, pngPath).Run()
			}
			os.Remove(svgPath)
			fmt.Printf("Сохранено в PNG: %s\n", pngPath)
		} else {
			fmt.Printf("Сохранено в SVG: %s\n", svgPath)
		}
		return
	}

	fileName := fmt.Sprintf("%s_%d.%s", result.Type, time.Now().Unix(), outputFormat)
	fullPath := filepath.Join(saveDir, fileName)
	f, _ := os.Create(fullPath)
	render.Render(ctx, result, f)
	f.Close()
	fmt.Printf("Сохранено: %s\n", fullPath)
}
