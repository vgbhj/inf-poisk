# Парсер

Парсер для обкачки статей с HLTV.org и Cybersport.ru с поддержкой MongoDB.

### С YAML конфигом

Создайте `config.yaml`:

```yaml
db:
  uri: "mongodb://localhost:27017"
  database: "crawler_db"
  collection: "documents"
logic:
  delay_between_pages: 500
  re_crawl_interval: 86400
workers: 4
site: "both"
```

Запуск:
```bash
go run main.go config.yaml
```

### Старый режим (флаги)

```bash
go run main.go -site both -workers 4
```

### Добавление существующих файлов в БД

```bash
go run main.go add-to-db -config config.yaml -source hltv
go run main.go add-to-db -config config.yaml -source cybersport
```

## Флаги (старый режим)

- `-site` - сайт: hltv, cybersport, both
- `-workers` - количество воркеров
- `-b` - использовать браузер
- `-collect-only` - только собрать ссылки
- `-download-only` - только скачать из CSV

