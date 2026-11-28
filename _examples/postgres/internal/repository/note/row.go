package note

import (
	"time"

	"app.internal/internal/entity"
)

type rowEntry struct {
	ID        int64     `db:"id"`
	Text      string    `db:"text"`
	CreatedAt time.Time `db:"created_at"`
}

func (r *rowEntry) toEntity() *entity.Note {
	return &entity.Note{
		ID:        new(entity.NoteID(r.ID)),
		Text:      r.Text,
		CreatedAt: new(r.CreatedAt),
	}
}
