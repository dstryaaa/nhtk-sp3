# loganalyzer — CLI-анализатор больших журналов событий

CLI-приложение на Go для анализа журналов событий: потоковое чтение, фильтрация,
агрегация и вывод сводки в терминал, JSON или CSV. Не загружает файл в память
целиком, обрабатывает повреждённые строки с указанием номера, даёт
детерминированный результат.

## Бизнес-требования

Службе поддержки нужно быстро находить частые ошибки и их источники в журналах
нескольких компонентов. Приложение должно:

- принимать путь к файлу и параметры через CLI;
- читать файл построчно (без загрузки всего файла в память);
- фильтровать по временному интервалу (`from` включительно, `to` исключительно),
  уровню (`level`) и источнику (`source`), а также их сочетанию;
- считать корректные и ошибочные строки, события после фильтрации,
  сводки по уровням и источникам, top-N кодов ошибок;
- выводить сводку в терминал и сохранять JSON или CSV;
- обрабатывать пустой и отсутствующий файл понятным образом;
- показывать номер ошибочной строки;
- давать детерминированный результат (сортировка при равных счётчиках).

**Формат входа:** `timestamp | level | source | eventCode | message`
- `timestamp` — ISO 8601 UTC (RFC3339, например `2026-09-01T00:08:59Z`);
- `level` — `TRACE`, `DEBUG`, `INFO`, `WARN`, `ERROR` (строго верхний регистр);
- `source`, `eventCode` — непустые;
- `message` — может быть пустым;
- `|` внутри `message` **допускается** (разбор через `SplitN(..., 5)`).

## Реализованный функционал

| Модуль | Назначение | Покрытие |
|--------|-----------|---------|
| `internal/parser` | Разбор одной строки в `Record`, ошибка с номером строки | 88.2% |
| `internal/reader` | Потоковое чтение файла (`bufio.Reader`), счётчики, лимит на битые строки | 93.9% |
| `internal/filter` | Фильтр по времени/уровню/источнику (AND) | 100% |
| `internal/agg` | Счётчики по уровням/источникам, top-N кодов ошибок, детерминизм | 100% |
| `internal/report` | Сборка отчёта, вывод в текст/JSON/CSV | 80.0% |
| `internal/cli` | Флаги, валидация, оркестрация всех слоёв | 94.4% |
| `cmd/loganalyzer` | `main()` — тонкая обёртка над `cli.Run` | 0% (не тестируется) |

**Общее покрытие: 87.5%** (порог ТЗ — 80%).

Непокрытые строки:

- `cmd/loganalyzer/main.go` — «клей», вызывает `cli.Run` и `os.Exit`; не тестируется намеренно.
- `internal/report/render.go` `WriteText` (61.7%) — ветки `if err != nil { return err }`
  после каждого `Fprintf`. В тестах используется `bytes.Buffer`, который ошибок не
  возвращает. В реальности эти ветки срабатывают только при ошибке записи
  (диск заполнен, pipe закрыт). Покрытие этих веток потребовало бы искусственного
  `failingWriter` с падением на каждом вызове — тест ради покрытия, а не ради логики.
- `internal/parser` `ParseError.Error`/`Unwrap` — вызываются косвенно через
  `fmt.Errorf("%v", err)` и `errors.As`, но `-func` их не засчитывает.

## CLI-синтаксис

```
loganalyzer -input <path> [-from <RFC3339>] [-to <RFC3339>] [-level <LEVEL>]
            [-source <source>] [-top <N>] [-format text|json|csv] [-output <path>]
```

| Флаг | По умолчанию | Описание |
|------|--------------|----------|
| `-input` | — (обязателен) | Путь к файлу журнала |
| `-from` | — | Начало интервала, RFC3339, **включительно** |
| `-to` | — | Конец интервала, RFC3339, **исключительно** |
| `-level` | — | Фильтр по уровню (строгий регистр) |
| `-source` | — | Фильтр по источнику (строгий регистр) |
| `-top` | `5` | Top-N кодов ошибок (`0` = все) |
| `-format` | `text` | Формат вывода: `text`, `json`, `csv` |
| `-output` | — (stdout) | Файл для записи отчёта |

### Примеры

**1. Полная сводка в терминал:**

```
./loganalyzer -input logs_events_sample.log
```

**2. Только ошибки gateway за интервал, top-3, JSON в файл:**

```
./loganalyzer -input logs_events_sample.log \
  -from 2026-09-01T00:00:00Z -to 2026-09-02T00:00:00Z \
  -level ERROR -source gateway -top 3 \
  -format json -output report.json
```

**3. CSV в stdout, все коды ошибок:**

```
./loganalyzer -input logs_events_sample.log -level ERROR -top 0 -format csv
```

## Развёртывание (WSL)

Проект разработан и проверен в WSL (Ubuntu). Все команды выполняются в терминале WSL.

1. **Установить Go** (если не установлен):
   ```
   sudo apt update && sudo apt install -y golang-go
   go version   # ожидается go1.23+
   ```
2. **Клонировать репозиторий:**
   ```
   git clone https://github.com/dstryaaa/nhtk-sp3.git
   cd nhtk-sp3
   ```
3. **Собрать:**
   ```
   go build -o loganalyzer ./cmd/loganalyzer
   ```
4. **Запустить** (см. примеры выше).

## Тестирование

Одна команда — тесты, покрытие, HTML-отчёт:

```
./run_tests.sh
```

Скрипт:

- запускает `go test ./... -coverprofile=coverage.out`;
- печатает покрытие по функциям (`go tool cover -func`);
- генерирует `coverage.html`;
- печатает путь к HTML-отчёту.

**HTML-отчёт:** `file:///home/semenov-n-m/nhtk-sp3/coverage.html`
(из Windows: `\\wsl$\Ubuntu\home\semenov-n-m\nhtk-sp3\coverage.html`).

Отдельные команды:

```
go test ./... -v                 # все тесты подробно
go test ./... -cover             # покрытие по пакетам
go test ./internal/parser -v     # один пакет
```

## Сборка и запуск

```
# сборка
go build -o loganalyzer ./cmd/loganalyzer

# запуск
./loganalyzer -input <path> [флаги]

# проверка синтаксиса без сборки
go vet ./...
```

## Структура проекта

```
.
├── cmd/loganalyzer/main.go     # main(): cli.Run + os.Exit
├── internal/
│   ├── parser/                 # разбор строки → Record
│   ├── reader/                 # потоковое чтение файла
│   ├── filter/                 # предикат по Record
│   ├── agg/                    # счётчики + top-N
│   ├── report/                 # сборка отчёта + вывод text/json/csv
│   └── cli/                    # флаги, валидация, оркестрация
├── go.mod
├── run_tests.sh                # тесты + coverage.html одной командой
├── .gitignore
└── README.md
```

## Результат на сэмпле

На `logs_events_sample.log` (1209 строк):

```
Total lines: 1209
Valid lines: 1202
Invalid lines: 6
Skipped blank: 1
Matched: 1202

By level: DEBUG 139, ERROR 477, INFO 437, WARN 149
By source: auth-api 251, gateway 113, inventory-api 317, orders-api 286, payment-worker 235
Top errors: PAYMENT_TIMEOUT 98, ORDER_REJECTED 88, HTTP_502 87, PAYMENT_DECLINED 83, STOCK_LOW 24
```

Повреждённые строки (с номерами): 36, 161, 281, 411, 551, 831.