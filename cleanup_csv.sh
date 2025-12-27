#!/bin/bash

CSV_FILE="corpus/hltv_links.csv"
HLTV_DIR="corpus/hltv"
BACKUP_FILE="${CSV_FILE}.backup"

# Проверяем существование файлов
if [ ! -f "$CSV_FILE" ]; then
    echo "Error: $CSV_FILE not found"
    exit 1
fi

if [ ! -d "$HLTV_DIR" ]; then
    echo "Error: $HLTV_DIR directory not found"
    exit 1
fi

# Создаем резервную копию
cp "$CSV_FILE" "$BACKUP_FILE"
echo "Backup created: $BACKUP_FILE"

# Считаем скачанные файлы
DOWNLOADED_COUNT=$(find "$HLTV_DIR" -maxdepth 1 -type f -name "*.txt" | wc -l)
echo "Found $DOWNLOADED_COUNT downloaded articles"

# Создаем временный файл
TEMP_FILE="${CSV_FILE}.tmp"

# Копируем header
head -1 "$CSV_FILE" > "$TEMP_FILE"

# Обрабатываем остальные строки
REMOVED=0
REMAINING=0

tail -n +2 "$CSV_FILE" | while IFS=',' read -r id slug; do
    # Удаляем кавычки если они есть
    id=$(echo "$id" | tr -d '"')
    
    # Проверяем существует ли файл с этим ID
    if [ ! -f "${HLTV_DIR}/${id}.txt" ]; then
        # Если файл не существует, добавляем строку обратно
        echo "$id,$slug" >> "$TEMP_FILE"
    fi
done

# Заменяем оригинальный файл
mv "$TEMP_FILE" "$CSV_FILE"

# Считаем оставшиеся строки (минус header)
REMAINING=$(tail -n +2 "$CSV_FILE" | wc -l)
REMOVED=$((DOWNLOADED_COUNT))

echo "Removed $REMOVED already downloaded articles from CSV"
echo "Remaining articles: $REMAINING"
