// pkg/output/json.go
package output

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/damirmur/swisseph_build/pkg/astro"
)

// JSONRenderer отвечает за сериализацию результатов в формат JSON/JS
type JSONRenderer struct{}

// Render принимает структуру AstroResult и потоково записывает её в io.Writer
func (r *JSONRenderer) Render(ctx context.Context, result *astro.AstroResult, w io.Writer) error {
	if result == nil {
		return fmt.Errorf("данные для рендеринга отсутствуют (результат равен nil)")
	}

	// Проверяем контекст на случай отмены операции пользователем в PicoClaw
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Настройка кодировщика
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	// РАЗВЕТВЛЕНИЕ: компактный JS только для календаря, для остального — стандартный JSON
	switch result.Type {
	case "calendar":
		// Формируем самоописываемую огибающую ответа календаря
		resp := astro.NewCalendarResponse(result.Events)

		// 4. Записываем префикс JavaScript (для совместимости генератора статики)
		prefix := "window.calendarResponse = "
		if _, err := io.WriteString(w, prefix); err != nil {
			return fmt.Errorf("ошибка записи JS-префикса: %w", err)
		}

		// 5. Кодируем огибающую {schema, meta, data} целиком
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(resp); err != nil {
			return fmt.Errorf("ошибка кодирования ответа календаря: %w", err)
		}
	default:
		// Для типов "natal", "synastry", "period" и любых других:
		// Потоковая запись оригинальной структуры напрямую в дескриптор файла
		if err := encoder.Encode(result); err != nil {
			return fmt.Errorf("ошибка потокового кодирования стандартного JSON: %w", err)
		}
	}

	return nil
}
