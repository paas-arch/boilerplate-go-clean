Правила — раздел «repository и entity» правила `.claude/rules/go-clean.md`. Образец — `_examples/postgres/internal/repository/note`. Базы в сервисе ещё нет — сначала `database.md`.

1. **Entity** `internal/entity/<name>.go`: ID-тип, enum-типы, структура с указателями на генерируемые БД поля. Готов, когда каждая колонка таблицы имеет поле.
2. **row.go**:

```go
package <name>

type rowEntry struct {
	ID        int64     `db:"id"`
	Text      string    `db:"text"`
	CreatedAt time.Time `db:"created_at"`
}

func (r *rowEntry) toEntity() *entity.<Name> {
	return &entity.<Name>{
		ID:        new(entity.<Name>ID(r.ID)),
		Text:      r.Text,
		CreatedAt: new(r.CreatedAt),
	}
}
```

3. **repository.go**:

```go
package <name>

const tableName = "<table>"

var (
	tableFields   = []string{"id", "text", "created_at"}
	selectBuilder = sq.Select(tableFields...).From(tableName).PlaceholderFormat(sq.Dollar)
	insertBuilder = sq.Insert(tableName).PlaceholderFormat(sq.Dollar)
)

type Repository struct {
	db     *sqlx.DB
	getter *trmsqlx.CtxGetter
}

func New(db *sqlx.DB, getter *trmsqlx.CtxGetter) *Repository {
	return &Repository{db, getter}
}

func (r *Repository) GetByName(ctx context.Context, name string) (*entity.<Name>, error) {
	query, args, err := selectBuilder.Where(sq.Eq{"name": name}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select: %w", err)
	}

	var row rowEntry
	if err := r.getter.DefaultTrOrDB(ctx, r.db).GetContext(ctx, &row, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to execute select: %w", err)
	}

	return row.toEntity(), nil
}
```

   Create — `insertBuilder.SetMap(...).Suffix("RETURNING id, created_at")` + `QueryRowxContext(...).Scan`, затем проставь `ID`/`CreatedAt` в entity. Уникальное поле — `ON CONFLICT (...) DO NOTHING RETURNING id` и `(false, nil)` на `sql.ErrNoRows`. Update/Delete — `ExecContext` и `RowsAffected() > 0`.
4. Подключи в `cmd/api/main.go` (и в воркерах, которым нужна таблица) в блоке `// Repository`: `<camel>Repo := <camel>Repository.New(db, trmsqlx.DefaultCtxGetter)`.
5. `go build ./...` проходит. Готов, когда каждый метод используется сервисом (или сервис создаётся следующим шагом — `service.md`).
