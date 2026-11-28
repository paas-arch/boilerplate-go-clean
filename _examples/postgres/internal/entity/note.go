package entity

import "time"

type NoteID int64

// Note — заметка, пример сущности; замени на свою.
type Note struct {
	ID        *NoteID
	Text      string
	CreatedAt *time.Time
}
