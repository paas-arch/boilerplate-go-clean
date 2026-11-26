# Architecture

Бэкенд сервиса: API и воркеры из одного репозитория, общая база Postgres. Собирается и выкатывается панелью paas по договору из `CLAUDE.md`.

## Directory Structure

```
.
├── api/schema.yaml                 # Контракт API (OpenAPI 3.0.3)
├── cmd/
│   ├── api/main.go                 # API: logger → config → база → repositories → services → usecases → handlers → webserver
│   └── workers/<имя>/main.go       # Воркеры; каждый — отдельный образ и процесс
├── internal/
│   ├── api/
│   │   ├── handler/<operation>/    # HTTP-ручки, пакет на операцию
│   │   └── mapper/                 # entity → DTO
│   ├── client/<provider>/          # Клиенты внешних API (пока нет)
│   ├── config/                     # Настройки из окружения
│   ├── entity/                     # Доменные типы
│   ├── generated/                  # Только кодогенерация, руками не правим
│   ├── model/                      # Типы внешних систем и входные DTO usecase
│   ├── repository/<table>/         # Запросы к базе, пакет на таблицу
│   ├── service/<mechanism>/        # Доменные механизмы
│   ├── usecase/<verb_noun>/        # Сценарии
│   └── util/
│       ├── logger/                 # Журнал в stdout (JSON), необязательный Sentry; вызов: log.Info(ctx, "сообщение", "ключ", значение, …), error — полем error
│       ├── psql/                   # Подключение к Postgres
│       └── webserver/              # HTTP-сервер, /health, codegen OpenAPI → Go
├── migrations/                     # Миграции goose; образ migrator применяет их перед выкаткой
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

Каждый пуш в `main` собирает образы с тегом `main-<номер сборки>-<7 знаков коммита>`; в панели версия видна как `#<номер>`. При раскатке панель сначала запускает `migrator` — он применяет новые миграции; если миграция упала, API и воркеры остаются на прежней версии. Затем запускаются API и воркеры новой версии, прежние получают SIGTERM.

Состав сервиса панель берёт из кода выкатываемой версии: папки `cmd/workers/*`, наличие `migrations/`, `api/schema.yaml`.

## Database Schema

| Таблица | Колонки | Заметки |
|---|---|---|
| `notes` | `id`, `text`, `created_at` | Пример; индекс по `created_at` для списка от новых к старым |

## API Endpoints

Все пути — `/api/v1/<группа>.<действие>`. Методы только `GET`, `POST`, `DELETE`; у `POST` параметры в JSON-теле.

| Method | Path | Operation |
|---|---|---|
| GET | `/health` | — (встроенная) |
| GET | `/api/v1/notes.get` | `notesGet` |
| POST | `/api/v1/notes.create` | `notesCreatePost` |

## Configuration

API слушает порт 8080 (`config.Port`): панель ждёт сервис на нём, переменной для порта нет.

| Переменная | Назначение |
|---|---|
| `APP_ENV` | `local` или `production`; в `production` не читается `.env` |
| `APP_HOST` | Адрес, на котором слушает API; пусто — все |
| `DATABASE_DSN` | Адрес базы с паролем. На сервере задаёт панель, когда базу подключили; пусто — базы нет |
| `SENTRY_DSN` | Необязательно |
| `TRUSTED_PROXY_CIDRS` | Прокси, чьим заголовкам с адресом клиента можно верить |
