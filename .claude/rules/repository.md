---
paths:
  - "internal/repository/**"
  - "internal/entity/**"
---

# Слой repository и entity

- Пакет `internal/repository/<table_singular>` (`note`): `repository.go` + `row.go`. Без `deps.go`, моков и юнит-тестов.
- `New(db *sqlx.DB, getter *trmsqlx.CtxGetter)`; каждый запрос идёт через `r.getter.DefaultTrOrDB(ctx, r.db)`, чтобы подхватить транзакцию из контекста. Саму транзакцию открывает usecase через менеджер транзакций (`tx.Do`).
- SQL — только `squirrel` c `PlaceholderFormat(sq.Dollar)`. На уровне пакета: `tableName`, `tableFields`, `selectBuilder`/`insertBuilder`/`updateBuilder`.
- `row.go`: `rowEntry` с тегами `db`, метод `toEntity()`. Наружу репозиторий отдаёт только `*entity.X`, строки БД не утекают.
- `Get*` без результата — `(nil, nil)` (`sql.ErrNoRows` гасится). Update/Delete возвращают `(bool, error)` по `RowsAffected`. `Create` заполняет `ID`/`CreatedAt` через `RETURNING`.
- Занятое уникальное значение — `ON CONFLICT (...) DO NOTHING RETURNING id` и `(false, nil)` из `Create`, а не отдельная проверка перед вставкой: её обгонит параллельный запрос.
- Список без предела не отдаём: `Limit` из константы сервиса.
- Время — `time.Now().UTC()`. Ошибки — `failed to build <op>` / `failed to execute <op>`.
- entity (`internal/entity/<name>.go`): ID-тип `type XID int64`, enum — `type XType string` + константы, nullable и генерируемые БД поля — указатели (`ID *XID`, `CreatedAt *time.Time`). Указатель на значение — встроенным `new(выражение)`.
- Репозиторий — одна таблица. Склейку данных из двух таблиц делает service.
