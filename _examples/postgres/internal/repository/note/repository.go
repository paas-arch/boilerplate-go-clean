package note

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	trmsqlx "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"
	"github.com/jmoiron/sqlx"

	"app.internal/internal/entity"
)

const tableName = "notes"

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

func (r *Repository) ListLatest(ctx context.Context, limit uint64) ([]*entity.Note, error) {
	query, args, err := selectBuilder.OrderBy("created_at DESC", "id DESC").Limit(limit).ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select: %w", err)
	}

	var rows []rowEntry
	if err := r.getter.DefaultTrOrDB(ctx, r.db).SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("failed to execute select: %w", err)
	}

	result := make([]*entity.Note, 0, len(rows))
	for i := range rows {
		result = append(result, rows[i].toEntity())
	}

	return result, nil
}

func (r *Repository) Count(ctx context.Context) (int64, error) {
	query, args, err := sq.Select("COUNT(*)").From(tableName).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build select: %w", err)
	}

	var n int64
	if err := r.getter.DefaultTrOrDB(ctx, r.db).GetContext(ctx, &n, query, args...); err != nil {
		return 0, fmt.Errorf("failed to execute select: %w", err)
	}

	return n, nil
}

// Create вставляет заметку и заполняет ID и время.
func (r *Repository) Create(ctx context.Context, note *entity.Note) error {
	query, args, err := insertBuilder.SetMap(map[string]any{
		"text":       note.Text,
		"created_at": time.Now().UTC(),
	}).Suffix("RETURNING id, created_at").ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert: %w", err)
	}

	var (
		id        int64
		createdAt time.Time
	)
	if err := r.getter.DefaultTrOrDB(ctx, r.db).QueryRowxContext(ctx, query, args...).Scan(&id, &createdAt); err != nil {
		return fmt.Errorf("failed to execute insert: %w", err)
	}

	note.ID = new(entity.NoteID(id))
	note.CreatedAt = &createdAt

	return nil
}
