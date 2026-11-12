package mapper

import (
	"time"

	"app.internal/internal/entity"
	"app.internal/internal/generated/api/schema/components"
)

func MapNotes(notes []*entity.Note) components.NotesResponse {
	result := make([]components.Note, 0, len(notes))
	for _, n := range notes {
		result = append(result, MapNote(n))
	}
	return components.NotesResponse{Notes: result}
}

func MapNote(n *entity.Note) components.Note {
	result := components.Note{Text: n.Text}
	if n.ID != nil {
		result.Id = int64(*n.ID)
	}
	if n.CreatedAt != nil {
		result.CreatedAt = n.CreatedAt.UTC().Format(time.RFC3339)
	}
	return result
}
