package storage

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

type StorageManager struct {
	dataDir string
	maxAge  time.Duration
}

// New инициализирует менеджер хранилища с указанием папки и срока жизни файлов
func New(dataDir string, retentionDays int) *StorageManager {
	return &StorageManager{
		dataDir: dataDir,
		maxAge:  time.Duration(retentionDays) * 24 * time.Hour,
	}
}

// StartAutoCleanup запускает фоновую горутину для очистки старых файлов
func (s *StorageManager) StartAutoCleanup(ctx context.Context, interval time.Duration) {
	// Создаем папку сразу, чтобы избежать ошибок при первом сканировании
	_ = os.MkdirAll(s.dataDir, 0755)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("[Storage] Фоновая очистка остановлена.")
				return
			case <-ticker.C:
				if err := s.Cleanup(); err != nil {
					log.Printf("[Storage] Ошибка при очистке диска: %v\n", err)
				}
			}
		}
	}()
}

// Cleanup сканирует директорию и удаляет файлы, срок хранения которых истек
func (s *StorageManager) Cleanup() error {
	now := time.Now()
	deletedCount := 0

	err := filepath.Walk(s.dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Пропускаем саму корневую директорию
		if path == s.dataDir {
			return nil
		}

		// Проверяем только регулярные файлы (json, txt, svg, png)
		if !info.IsDir() {
			if now.Sub(info.ModTime()) > s.maxAge {
				if err := os.Remove(path); err != nil {
					return fmt.Errorf("не удалось удалить файл %s: %w", path, err)
				}
				deletedCount++
			}
		}
		return nil
	})

	if deletedCount > 0 {
		log.Printf("[Storage] Ротация завершена. Удалено устаревших файлов: %d\n", deletedCount)
	}
	return err
}
