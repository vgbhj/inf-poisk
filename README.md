# Информационный поиск

## Реализовано

1. Добыча корпуса - `main.go` (5000+ документов о киберспорте)
2. Поисковый робот - `parser/` (парсинг с HLTV.org и Cybersport.ru)
3. Токенизация - `tokenizer/tokenizer.cpp` (1млн+ токенов)
4. Стемминг - `stemmer/stemmer.cpp` (100k+ уникальных слов)
5. Закон Ципфа - `zipf/frequency.cpp` (анализ частотности)
6. Булев индекс - `search/` (инвертированный индекс)
7. Булев поиск - `search/` (AND, OR, NOT запросы)

## Использование

```bash
cd search
make clean && make

# Построить индекс
./index_builder ../corpus/cybersport/parsed index.idx

# Поиск
./searcher index.idx "cs2 and турнир"
./searcher index.idx "победа or матч"
./searcher index.idx "-хейтер информация"

# Статистика
./index_stats index.idx
```

## Структура

```
├── main.go              # Парсер на Go
├── parser/              # Логика парсирования
├── corpus/              # Скачанные документы
├── tokenizer/           # Токенизация (C++)
├── stemmer/             # Стемминг (C++)
├── zipf/                # Анализ Ципфа (C++)
└── search/              # Булев поиск (C++)
```

## Компоненты поиска

Структуры данных без STL:
- `vector.h`, `string.h`, `hashmap.h`, `posting_list.h`

Основной код:
- `boolean_index.h` - инвертированный индекс
- `query_parser.h` - парсер запросов
- `boolean_searcher.h` - выполнение поиска

Утилиты:
- `index_builder` - создание индекса
- `searcher` - поиск
- `index_stats` - статистика
