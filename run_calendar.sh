#!/bin/bash

# Переходим в директорию проекта
cd /home/dmyr/swisseph_build/ || exit 1

# Принимаем аргументы или берём текущие значения
INPUT_YEAR=${1:-$(date +%Y)}
INPUT_MONTH=${2:-$(date +%-m)}

# --- Формат вывода ---
FORMAT="${3:-console}"
case "$FORMAT" in
    console|json|text|image) ;; # допустимые значения
    *)
        echo "Ошибка: Неверный формат '$FORMAT'. Возможные: console, json, text, image."
        exit 1
        ;;
esac

# --- ВАЛИДАЦИЯ ДАННЫХ ---

# Проверка года: должен состоять ровно из 4 цифр
if [[ ! "$INPUT_YEAR" =~ ^[0-9]{4}$ ]]; then
    echo "Ошибка: Год должен быть четырехзначным числом (например, 2026)."
    exit 1
fi

# Проверка месяца: должен быть числом от 1 до 12, 0 -год
# Убираем ведущий ноль, если он есть, для корректного сравнения
CLEAN_MONTH=$((10#$INPUT_MONTH))
if (( CLEAN_MONTH < 0 || CLEAN_MONTH > 12 )); then
    echo "Ошибка: Месяц должен быть числом от 1 до 12."
    exit 1
fi

# Если валидация прошла, фиксируем переменные
CURRENT_YEAR="$INPUT_YEAR"
CURRENT_MONTH="$CLEAN_MONTH"

# Определяем имя месяца для красивого вывода
MONTH_NAME=$(date -d "2000-$CURRENT_MONTH-01" +%B 2>/dev/null || date +%B)

# --- ПРОВЕРКА АКТУАЛЬНОСТИ БИНАРНИКА ---
NEED_BUILD=false

if [ ! -f "./astro" ]; then
    NEED_BUILD=true
else
    # Находим самый свежий файл .go
    LATEST_SOURCE=$(find . -name "*.go" -type f -printf '%T@ %p\n' 2>/dev/null | sort -n | tail -1 | cut -d' ' -f2-)

    if [ -n "$LATEST_SOURCE" ] && [ "$LATEST_SOURCE" -nt "./astro" ]; then
        echo "Исходный код изменился ($LATEST_SOURCE новее, чем ./astro)."
        NEED_BUILD=true
    fi
fi

# --- КОМПИЛЯЦИЯ ---
if [ "$NEED_BUILD" = true ]; then
    echo "Компиляция проекта в ./astro..."
    go build -o astro ./cmd/astro/main.go

    if [ $? -ne 0 ]; then
        echo "Ошибка компиляции!"
        exit 1
    fi
    echo "Компиляция успешна!"
else
    echo "Бинарный файл ./astro актуален. Компиляция пропущена."
fi

# --- ЗАПУСК ---
echo "Запуск генерации календаря на $MONTH_NAME $CURRENT_YEAR..."
echo "------------------------------------------------"
./astro calendar --year "$CURRENT_YEAR" --month "$CURRENT_MONTH" \
    --format "${FORMAT}"
