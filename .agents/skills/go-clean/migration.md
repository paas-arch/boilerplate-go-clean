Правила — раздел «Миграции» правила `.claude/rules/go-clean.md`. Базы в сервисе ещё нет — сначала `database.md`.

1. Создай файл `migrations/NNNNN_<create_x|add_x_to_y>.sql` со следующим номером. Одна таблица — один файл: несколько таблиц — несколько файлов в порядке FK.
2. Заполни только Up:

```sql
-- +goose Up
-- +goose StatementBegin
CREATE TABLE <table> (
    id          bigserial  PRIMARY KEY,
    owner_id    bigint     NOT NULL REFERENCES owners (id) ON DELETE CASCADE,
    created_at  timestamp  NOT NULL DEFAULT NOW(),
    updated_at  timestamp  NOT NULL DEFAULT NOW()
);

-- <какой запрос обслуживает индекс>
CREATE INDEX idx_<table>_owner_id ON <table> (owner_id);
-- +goose StatementEnd
```

3. Проверь: `make db.up migrate.up && make run` — сервис стартует. Готов, когда новая версия есть в `goose_db_version`.
4. Обнови раздел «Database Schema» в `ARCHITECTURE.md`.

Дальше обычно — `repository.md`.
