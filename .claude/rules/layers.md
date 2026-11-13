# Слои сервиса

Поток вызова: `handler → usecase → service → repository → Postgres`; к внешнему API — `service → client → внешний API`. Подробности и «почему» — `ARCHITECTURE.md`.

- Зависимость идёт только вниз. Каждый пакет описывает нужное ему от соседей локальными интерфейсами в своём `deps.go`; конкретные типы встречаются только в `cmd/*/main.go`.
- Точки входа: `cmd/api` — API `/api/v1/<группа>.<действие>`; `cmd/workers/<имя>` — воркеры. Других папок в `cmd/` не заводим: сборка и панель их не увидят.
- Сборка — вручную в `main.go`, снизу вверх: база → clients → repositories → services → usecases → handlers → webserver. Импорт с алиасом `<camelName><Layer>` (`noteRepository`, `notebookService`), переменная — `<camelName><Repo|Cl|Svc|Case|Method>`.
- Пакет = одна единица: handler — одна операция OpenAPI, usecase — один сценарий с методом `Run`, service — один доменный механизм (`notebook`), client — один внешний API, repository — одна таблица.
- Handler зовёт service напрямую только для простого чтения без ветвлений. Валидация ввода, несколько сервисов или бизнес-ошибка — значит нужен usecase.
- Типы: `entity` — доменные модели (`Note`); `model` — структуры внешних систем, которые возвращают клиенты, и входные DTO usecase (`<Usecase>Params`); DTO — только сгенерированные, наружу через `internal/api/mapper`.
- Ошибки по слоям: client возвращает типизированный `APIError`; service оборачивает `fmt.Errorf("failed to <действие>: %w", err)` и объявляет sentinel `ErrXxx`; usecase переводит в `errx.Validation/NotFound/Forbidden/Internal` с `errx.WithPublicMessage("<по-русски>")`; handler переводит `errx` в HTTP-код.
- Имена функций и методов начинаются с глагола: `ListLatest`, `Add`, `parseTag`. Исключения: `New`, `Run`, `Handle`, опции и контекст `With*`, методы стандартных интерфейсов, сгенерированный код и `internal/util/webserver` (его API зашит в codegen).
- Секреты (пароли, токены, ключи) не попадают в логи и тексты ошибок.
- База: sqlx + lib/pq, подключение в `internal/util/psql`. Схему меняют только миграции goose в `migrations/`; применяет их образ `migrator` перед каждой выкаткой, сам сервис миграций не запускает.
- `internal/generated/` правит только `make codegen`.
- Юнит-тесты — только табличные: слайс кейсов + цикл с `t.Run(tt.name, ...)`, по одной таблице на метод. `testify` + `gomock`, моки из `deps.go` (`make generate`) в `_mocks/`. Имена кейсов — по-русски.
- Модуль Go — из первой строки `go.mod`, в шаблонах скилов он обозначен `<module>`.

Скилы на создание слоя: `new-client`, `new-service`, `new-usecase`, `new-handler`, `new-repository`, `new-migration`. Правила базы — `repository.md` и `migrations.md`.
