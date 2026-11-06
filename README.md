# go-newyear-api

[![CI](https://github.com/andreychirkov42/go-newyear-api/actions/workflows/ci.yml/badge.svg)](https://github.com/andreychirkov42/go-newyear-api/actions/workflows/ci.yml)

HTTP-сервис на Go, который считает, сколько календарных дней осталось от заданной
даты до ближайшего следующего Нового года (1 января следующего года).

- 31 декабря → `1` день, 1 января → `365` дней (`366` в високосном году).
- Високосные годы учитываются по правилам григорианского календаря
  (2024 и 2000 — високосные, 2023 и 2100 — нет).
- Время суток не влияет на результат, учитывается только дата.

## Требования

- Go **1.26** или новее (версия указана в `go.mod`);
- для запуска в контейнере — Docker (необязательно);
- внешних зависимостей нет, используется только стандартная библиотека.

## Структура проекта

```
cmd/server/          точка входа: HTTP-сервер, настройки, graceful shutdown
internal/newyear/    расчёт дней до Нового года (не зависит от HTTP)
internal/api/        HTTP-обработчики, JSON-ответы, логирование запросов
.github/workflows/   CI на GitHub Actions
Dockerfile           сборка образа с сервисом
```

## Сборка

```bash
go build -o server ./cmd/server
```

На Windows получится `server.exe`: `go build -o server.exe ./cmd/server`.

## Запуск

```bash
go run ./cmd/server
```

или собранный бинарник:

```bash
./server
```

Сервер слушает `:8080`. Адрес можно поменять переменной окружения `ADDR`:

```bash
ADDR=:9090 go run ./cmd/server
```

Сервис пишет логи в stdout в формате JSON (`log/slog`): запуск, остановка и
одна запись на каждый запрос (метод, путь, query, статус, длительность).
Останавливается по `Ctrl+C` / `SIGTERM`, дожидаясь завершения текущих запросов.

### В Docker

```bash
docker build -t go-newyear-api .
docker run --rm -p 8080:8080 go-newyear-api
```

## Тесты

```bash
go test ./...
```

| Команда | Что делает |
|---|---|
| `go test -v ./...` | все тесты с подробным выводом |
| `go test -cover ./...` | тесты + процент покрытия по пакетам |
| `go test -coverprofile=coverage.out ./...` и `go tool cover -html=coverage.out` | отчёт о покрытии в браузере |
| `go test -race ./...` | тесты с детектором гонок (нужен cgo: на Linux/macOS работает сразу, на Windows нужен gcc) |
| `go test -run='^$' -bench=. ./internal/newyear` | benchmark функции расчёта |

Что проверяется:

- **unit-тесты** (`internal/newyear`) — table-driven тест функции `DaysUntil`:
  начало и конец обычного и високосного года, даты до, в и после 29 февраля,
  2000 и 2100 годы, середина года; отдельные тесты на то, что время суток
  игнорируется и дата берётся в часовом поясе переданного времени;
- **интеграционные тесты** (`internal/api`) — поднимают настоящий HTTP-сервер
  через `httptest.NewServer`, отправляют запросы, проверяют статус, `Content-Type`,
  разбирают JSON и сверяют значения; ошибочный ввод, 404 и 405;
- **тест сервера** (`cmd/server`) — запуск на свободном порту, health check,
  корректная остановка и ошибка при занятом порту.

## HTTP API

Все ответы — JSON (`Content-Type: application/json; charset=utf-8`).

### `GET /api/v1/days-until-new-year`

Возвращает количество дней до следующего Нового года.

**Параметры запроса**

| Параметр | Обязательный | Формат | Описание |
|---|---|---|---|
| `date` | нет | `YYYY-MM-DD` (ISO 8601, например `2024-02-29`) | дата, от которой ведётся расчёт. Если параметра нет — берётся текущая дата сервера (в его часовом поясе; в Docker-образе это UTC) |

Принимается только формат `YYYY-MM-DD` с ведущими нулями: `2024-01-05` — правильно,
`2024-1-5`, `05.01.2024`, `2024-01-05T00:00:00Z` — ошибка. Несуществующие даты
(`2023-02-29`, `2024-13-01`) тоже считаются ошибкой. Параметр можно передать не больше одного раза.

**Ответ `200 OK`**

```bash
curl "http://localhost:8080/api/v1/days-until-new-year?date=2024-02-29"
```

```json
{"date":"2024-02-29","next_new_year":"2025-01-01","days_left":307}
```

| Поле | Тип | Описание |
|---|---|---|
| `date` | string, `YYYY-MM-DD` | дата, для которой выполнен расчёт |
| `next_new_year` | string, `YYYY-MM-DD` | дата ближайшего следующего Нового года |
| `days_left` | integer | количество дней до него (от 1 до 366) |

Без параметра:

```bash
curl "http://localhost:8080/api/v1/days-until-new-year"
```

```json
{"date":"2026-11-06","next_new_year":"2027-01-01","days_left":56}
```

**Ответ `400 Bad Request`** — некорректная дата:

```bash
curl "http://localhost:8080/api/v1/days-until-new-year?date=2024-13-01"
```

```json
{"error":"invalid date: expected format YYYY-MM-DD, for example 2024-02-29"}
```

### `GET /healthz`

Проверка состояния сервиса (для Docker, Kubernetes, балансировщиков).

```json
{"status":"ok"}
```

Почему такой формат: системы мониторинга смотрят в первую очередь на код
ответа (`200` — сервис жив), а маленький JSON с полем `status` оставляет ответ
машиночитаемым, как и остальной API, и его легко расширить (версия, проверки
зависимостей), не ломая клиентов. У сервиса нет внешних зависимостей, поэтому
если процесс отвечает — он работоспособен.

### Ошибки

| Код | Когда | Тело |
|---|---|---|
| `400 Bad Request` | неверный формат или несуществующая дата, повторённый `date` | `{"error":"..."}` |
| `404 Not Found` | неизвестный путь | `{"error":"not found"}` |
| `405 Method Not Allowed` | метод, отличный от `GET`/`HEAD`; в заголовке `Allow: GET, HEAD` | `{"error":"method not allowed"}` |

## CI

Workflow `.github/workflows/ci.yml` запускается при push в `main` и на каждый Pull Request.
Если любая проверка падает, весь workflow считается неуспешным.

| Job | Проверки |
|---|---|
| **Lint, build and test** | `gofmt -l` (неотформатированные файлы → ошибка), `go vet`, `go mod tidy -diff` (go.mod не соответствует коду → ошибка), `go mod verify`, `go build`, `go test -race` с покрытием, порог покрытия 80%, benchmark |
| **Vulnerability check** | `govulncheck ./...` — известные уязвимости в коде и стандартной библиотеке |
| **Docker image** | сборка образа, запуск контейнера и smoke-тест `/healthz` и API |

Отчёт о покрытии выводится в summary запуска и сохраняется артефактом `coverage`.

Те же проверки локально:

```bash
gofmt -l .
go vet ./...
go mod tidy -diff
go mod verify
go build ./...
go test -race -cover ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```
