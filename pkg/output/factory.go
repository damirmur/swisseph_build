// pkg/output/factory.go
package output

import "fmt"

func GetRenderer(format string) (Renderer, error) {
	switch format {
	case "json":
		return &JSONRenderer{}, nil
	// Кейсы для остальных форматов будут добавляться сюда
	case "console":
		return &ConsoleRenderer{}, nil // <-- Добавлено
	case "text":
		return &TextRenderer{}, nil // <-- Добавлено
	case "svg":
		return &ImageRenderer{ConvertToPNG: false}, nil
	case "png":
		return &ImageRenderer{ConvertToPNG: true}, nil
	default:
		return nil, fmt.Errorf("неизвестный формат вывода: %s", format)
	}
}
