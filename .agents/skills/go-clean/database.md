# База

Правила — разделы «repository и entity», «Миграции» и «paas.toml» правила `.claude/rules/go-clean.md`. Образец — `_examples/postgres/` (файлы лежат по путям, куда их копировать).

Делается один раз, когда сервису впервые нужна база. Дальше новые таблицы — `migration.md` и `repository.md`.

1. **Код из образца** — скопируй в сервис:
   - `_examples/postgres/internal/util/psql/` → `internal/util/psql/` (подключение, прячет пароль из ошибок; с тестом);
   - `_examples/postgres/docker-compose.yaml` → `docker-compose.yaml` (локальный Postgres на порту 5433);
   - цели из `_examples/postgres/Makefile.db` → в `Makefile`, и выполни строку `go get -tool ...` из его шапки.
2. **Зависимости:** `go get github.com/jmoiron/sqlx github.com/lib/pq github.com/Masterminds/squirrel github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2 github.com/avito-tech/go-transaction-manager/trm/v2`.
3. **`internal/config/app.go`** — поле, проверка и срок подключения:

```go
// DatabaseDSN задаёт панель, когда в paas.toml включена база; в журнал и ошибки целиком не попадает.
DatabaseDSN string `env:"DATABASE_DSN"`
```

```go
// DatabaseConnectTimeout — срок только на подключение: недоступная база не должна вешать запуск.
DatabaseConnectTimeout = 10 * time.Second
```

   В `Validate`:

```go
if c.DatabaseDSN == "" {
	errs = append(errs, errors.New("DATABASE_DSN is empty: enable [postgresql] in paas.toml and roll out the version"))
}
```

   Без адреса процесс не стартует: раскатка упадёт с этим текстом в журнале, а прежняя версия продолжит работать. Режима «работаю без базы» не делай. В `config_test.go` добавь кейс «нет адреса базы» с `wantErr: true`, остальным кейсам дай адрес.
4. **`cmd/api/main.go`** (и каждый воркер и крон, которым нужна база) — в начале сборки зависимостей:

```go
// Database
dbCtx, dbCancel := context.WithTimeout(ctx, config.DatabaseConnectTimeout)
db, err := psql.NewPostgres(dbCtx, cfg.DatabaseDSN)
dbCancel()
if err != nil {
	log.Error(ctx, fmt.Errorf("failed to connect to database: %w", err))
	return fail
}
defer func() { _ = db.Close() }()

// Транзакции — для сценария, который пишет в несколько таблиц: tx.Do(ctx, func(ctx) error {...}).
txManager, err := manager.New(trmsqlx.NewDefaultFactory(db))
if err != nil {
	log.Error(ctx, fmt.Errorf("failed to init transaction manager: %w", err))
	return fail
}
```

   Импорты: `trmsqlx "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"`, `"github.com/avito-tech/go-transaction-manager/trm/v2/manager"`, `"<module>/internal/util/psql"`. `txManager` передаётся в `New(...)` usecase, которому нужна транзакция; пока такого нет — не создавай его.
5. **`.env.example`:** `DATABASE_DSN=postgres://app:app@localhost:5433/app?sslmode=disable`.
6. **`paas.toml`:** `[postgresql] enabled = true`.
7. **Первая таблица** — `migration.md`, затем `repository.md` и `service.md`. Образец всей цепочки — `_examples/postgres/migrations/`, `internal/entity/note.go`, `internal/repository/note/`, `internal/service/notebook/`.
8. **Проверка:** `make db.up migrate.up && make run` — сервис стартует; без `DATABASE_DSN` в `.env` — не стартует и пишет, почему. Затем `check.md`.
9. **`ARCHITECTURE.md`:** раздел «Database Schema» с таблицей, `DATABASE_DSN` в «Configuration», `internal/util/psql/` и `internal/repository/` в дереве.

Скажи человеку: базу выдаст следующая раскатка этой версии в панели.
