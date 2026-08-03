package interactive

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type UserInput struct {
	City    string
	Date    string
	Time    string
	IsLocal bool
	Gender  string // "m", "f", or ""
}

type GeoData struct {
	City      string  `json:"city"`
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
	Timezone  string  `json:"timezone"`
	UTCOffset float64 `json:"offset_hours"`
}

func readLine(reader *bufio.Reader, prompt string) string {
	fmt.Print(prompt)
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

func parseDate(input string) (string, error) {
	re := regexp.MustCompile(`[\-:\./, ]+`)
	cleaned := re.ReplaceAllString(input, "-")

	t, err := time.Parse("2006-01-02", cleaned)
	if err == nil {
		return t.Format("2006-01-02"), nil
	}

	t, err = time.Parse("02-01-2006", cleaned)
	if err == nil {
		return t.Format("2006-01-02"), nil
	}

	return "", fmt.Errorf("неверный формат даты")
}

func parseTime(input string) (string, error) {
	re := regexp.MustCompile(`[\-:\./, ]+`)
	cleaned := re.ReplaceAllString(input, ":")

	t, err := time.Parse("15:04", cleaned)
	if err == nil {
		return t.Format("15:04"), nil
	}

	return "", fmt.Errorf("неверный формат времени")
}

func parseConfirm(input string) bool {
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" || input == "y" || input == "yes" || input == "д" || input == "да" || input == "true" {
		return true
	}
	return false
}

func parseGender(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "m" || input == "м" {
		return "m"
	}
	if input == "f" || input == "ж" {
		return "f"
	}
	return ""
}

func CollectData(allowEmptyDate bool) (*UserInput, error) {
	reader := bufio.NewReader(os.Stdin)
	ui := &UserInput{}

	for {
		ui.City = readLine(reader, "Введите город: ")
		if ui.City != "" {
			break
		}
		fmt.Println("Ошибка: город не может быть пустым.")
	}

	for {
		prompt := "Введите дату (YYYY-MM-DD или DD-MM-YYYY)"
		if allowEmptyDate {
			prompt += " [Enter - сейчас]"
		}
		prompt += ": "

		dateInput := readLine(reader, prompt)
		if dateInput == "" {
			if allowEmptyDate {
				now := time.Now()
				ui.Date = now.Format("2006-01-02")
				ui.Time = now.Format("15:04")
				ui.IsLocal = true
				break
			} else {
				fmt.Println("Ошибка: дата не может быть пустой для этой карты.")
				continue
			}
		}

		parsedDate, err := parseDate(dateInput)
		if err != nil {
			fmt.Println("Ошибка: неверный формат даты. Попробуйте еще раз.")
			continue
		}
		ui.Date = parsedDate

		for {
			timeInput := readLine(reader, "Введите время (HH:MM): ")
			if timeInput == "" {
				fmt.Println("Ошибка: время не может быть пустым.")
				continue
			}
			parsedTime, err := parseTime(timeInput)
			if err != nil {
				fmt.Println("Ошибка: неверный формат времени. Попробуйте еще раз.")
				continue
			}
			ui.Time = parsedTime
			break
		}

		localInput := readLine(reader, "Это местное время? (y/n) [Enter - да]: ")
		ui.IsLocal = parseConfirm(localInput)
		break
	}

	genderInput := readLine(reader, "Введите пол (m/м - мужской, f/ж - женский) [Enter - пропуск]: ")
	ui.Gender = parseGender(genderInput)

	return ui, nil
}

// GetGeoData запрашивает координаты и часовой пояс по названию населенного пункта онлайн.
func GetGeoData(input *UserInput) (*GeoData, error) {
	// 1. Поиск координат через OpenStreetMap Nominatim API
	escapedCity := url.QueryEscape(input.City)
	nominatimURL := fmt.Sprintf("https://nominatim.openstreetmap.org/search?q=%s&format=json&limit=1", escapedCity)

	req, err := http.NewRequest("GET", nominatimURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса к Nominatim: %v", err)
	}
	req.Header.Set("User-Agent", "picoclaw-astro-v2")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка сети при поиске города: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("неверный статус ответа геокодера: %d", resp.StatusCode)
	}

	var results []struct {
		Lat         string `json:"lat"`
		Lon         string `json:"lon"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("ошибка парсинга геоданных: %v", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("населенный пункт %q не найден", input.City)
	}

	lat, err := strconv.ParseFloat(results[0].Lat, 64)
	if err != nil {
		return nil, fmt.Errorf("ошибка преобразования широты: %v", err)
	}
	lon, err := strconv.ParseFloat(results[0].Lon, 64)
	if err != nil {
		return nil, fmt.Errorf("ошибка преобразования долготы: %v", err)
	}

	// 2. Получение часового пояса по координатам через TimeAPI
	timezoneURL := fmt.Sprintf("https://www.timeapi.io/api/TimeZone/coordinate?latitude=%.6f&longitude=%.6f", lat, lon)
	timeReq, err := http.NewRequest("GET", timezoneURL, nil)
	if err == nil {
		timeReq.Header.Set("User-Agent", "picoclaw-astro-v2")
		if timeResp, err := client.Do(timeReq); err == nil {
			defer timeResp.Body.Close()
			if timeResp.StatusCode == http.StatusOK {
				var tzResult struct {
					TimeZone        string `json:"timeZone"`
					CurrentUtcOffset struct {
						Seconds int `json:"seconds"`
					} `json:"currentUtcOffset"`
				}
				if err := json.NewDecoder(timeResp.Body).Decode(&tzResult); err == nil {
					return &GeoData{
						City:      input.City,
						Latitude:  lat,
						Longitude: lon,
						Timezone:  tzResult.TimeZone,
						UTCOffset: float64(tzResult.CurrentUtcOffset.Seconds) / 3600.0,
					}, nil
				}
			}
		}
	}

	// Дефолтный фолбек на UTC+3 (например, Москва), если Timezone API недоступен
	return &GeoData{
		City:      input.City,
		Latitude:  lat,
		Longitude: lon,
		Timezone:  "Europe/Moscow",
		UTCOffset: 3.0,
	}, nil
}

func ConfirmData(input *UserInput, gd *GeoData) bool {
	fmt.Printf("\n--- Подтверждение данных ---\n")
	fmt.Printf("Локация: %s (%.2f, %.2f)\n", gd.City, gd.Latitude, gd.Longitude)
	fmt.Printf("Время: %s %s (UTC%+g)\n", input.Date, input.Time, gd.UTCOffset)
	if input.Gender != "" {
		fmt.Printf("Пол: %s\n", input.Gender)
	}
	fmt.Print("\nВерно? (y/n) [Enter - да]: ")
	
	reader := bufio.NewReader(os.Stdin)
	confirm, _ := reader.ReadString('\n')
	return parseConfirm(confirm)
}

func MainMenu() int {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("\n=== Интерактивный режим ===")
	fmt.Println("1 - Натал (по умолчанию)")
	fmt.Println("2 - Синастрия")
	fmt.Println("3 - Календарь")
	
	for {
		choice := readLine(reader, "Выберите тип расчета [1-3, Enter - 1]: ")
		if choice == "" {
			return 1
		}
		val, err := strconv.Atoi(choice)
		if err == nil && val >= 1 && val <= 3 {
			return val
		}
		fmt.Println("Ошибка: введите число от 1 до 3.")
	}
}

func CollectCalendarData() (int, int, string) {
	reader := bufio.NewReader(os.Stdin)
	now := time.Now()
	
	year := now.Year()
	month := int(now.Month())

	yearInput := readLine(reader, fmt.Sprintf("Введите год [Enter - %d]: ", year))
	if yearInput != "" {
		if y, err := strconv.Atoi(yearInput); err == nil {
			year = y
		}
	}

	monthInput := readLine(reader, fmt.Sprintf("Введите месяц (1-12) или 0 для всего года [Enter - %d]: ", month))
	if monthInput != "" {
		if m, err := strconv.Atoi(monthInput); err == nil && m >= 0 && m <= 12 {
			month = m
		}
	}

	city := readLine(reader, "Введите город (или нажмите Enter для UTC+0): ")

	return year, month, city
}