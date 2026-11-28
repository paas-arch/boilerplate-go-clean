# Architecture

Бэкенд сервиса: API, а когда понадобятся — воркеры, кроны и база Postgres, из одного репозитория. Собирается и выкатывается панелью paas по договору из `CLAUDE.md`.

## Directory Structure

```
.
├── _examples/                      # Образцы базы, воркера и крона; Go их не собирает
├── api/schema.yaml                 # Контракт API (OpenAPI 3.0.3)
├── cmd/
│   ├── api/main.go                 # API: logger → config → (база → clients → repositories) → services → usecases → handlers → webserver
│   ├── workers/<имя>/main.go       # Воркеры, когда появятся; каждый — отдельный образ и процесс
│   └── cron/<имя>/main.go          # Кроны, когда появятся; один проход и выход
├── internal/
│   ├── api/
│   │   ├── handler/<operation>/    # HTTP-ручки, пакет на операцию
│   │   └── mapper/                 # entity → DTO
│   ├── client/<provider>/          # Клиенты внешних API (пока нет)
│   ├── config/                     # Настройки из окружения
│   ├── entity/                     # Доменные типы
│   ├── generated/                  # Только кодогенерация, руками не правим
│   ├── model/                      # Типы внешних систем и входные DTO usecase
│   ├── repository/<table>/         # Запросы к базе, пакет на таблицу (пока нет)
│   ├── service/<mechanism>/        # Доменные механизмы
│   ├── usecase/<verb_noun>/        # Сценарии
│   └── util/
│       ├── logger/                 # Журнал в stdout (JSON), необязательный Sentry; вызов: log.Info(ctx, "сообщение", "ключ", значение, …), error — полем error
│       └── webserver/              # HTTP-сервер, /health, codegen OpenAPI → Go
├── migrations/                     # Миграции goose (пока пусто, папка нужна сборке); образ migrator применяет их перед выкаткой
├── Dockerfile                      # Образы сервиса и migrator — не менять
└── .github/workflows/              # release.yml — сборка образов (не менять), ci.yml — проверки
```

## Request Lifecycle

```
HTTP Request
  → webserver (chi: request id, адрес клиента, recover)
  → generated/<op>/handle.go        разбор тела/параметров + проверка по схеме
  → api/handler/<op>                разбор DTO, перевод ошибок errx → HTTP
  → usecase/<name>                  валидация ввода, оркестрация, ошибки → errx
  → service/<name>                  доменная логика
  → repository/<table>              SQL через squirrel
```

## Выкатка

Каждый пуш в `main` собирает образы с тегом `main-<номер сборки>-<7 знаков коммита>`; в панели версия видна как `#<номер>`. При раскатке панель сначала запускает `migrator` — он применяет новые миграции; если миграция упала, процессы остаются на прежней версии. Миграции идут, только когда в `paas.toml` включена база. Затем запускаются API и воркеры новой версии, прежние получают SIGTERM; кроны новой версии запускаются по расписанию.

Состав сервиса панель берёт из `paas.toml` и кода выкатываемой версии: копии и размер, база, переменные, записи `[[workers]]` и `[[crons]]` вместе с папками `cmd/workers/*` и `cmd/cron/*`, контракт `api/schema.yaml`.

## Database Schema

Базы пока нет. Когда появится — таблица на каждую таблицу базы: имя, колонки, заметки об индексах.

## API Endpoints

Все пути — `/api/v1/<группа>.<действие>`. Методы только `GET`, `POST`, `DELETE`; у `POST` параметры в JSON-теле.

| Method | Path | Operation |
|---|---|---|
| GET | `/health` | — (встроенная) |
| POST | `/api/v1/greetings.create` | `greetingsCreatePost` |

## Configuration

API слушает порт 8080 (`config.Port`): панель ждёт сервис на нём, переменной для порта нет.

| Переменная | Назначение |
|---|---|
| `APP_ENV` | `local` или `production`; в `production` не читается `.env` |
| `APP_HOST` | Адрес, на котором слушает API; пусто — все |
| `SENTRY_DSN` | Необязательно |
| `TRUSTED_PROXY_CIDRS` | Прокси, чьим заголовкам с адресом клиента можно верить |
