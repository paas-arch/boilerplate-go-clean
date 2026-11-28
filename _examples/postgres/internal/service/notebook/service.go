// Package notebook — образец сервиса над таблицей: прячет от сценариев, как заметки хранятся.
package notebook

import (
	"context"
	"fmt"

	"app.internal/internal/entity"
)

// latestLimit — сколько заметок отдаёт список: без предела ответ рос бы вместе с таблицей.
const latestLimit = 100

type Service struct {
	notes noteRepository
}

func New(notes noteRepository) *Service {
	return &Service{notes: notes}
}

func (s *Service) ListLatest(ctx context.Context) ([]*entity.Note, error) {
	notes, err := s.notes.ListLatest(ctx, latestLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to list notes: %w", err)
	}
	return notes, nil
}

func (s *Service) Count(ctx context.Context) (int64, error) {
	n, err := s.notes.Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count notes: %w", err)
	}
	return n, nil
}

func (s *Service) Add(ctx context.Context, text string) (*entity.Note, error) {
	note := &entity.Note{Text: text}
	if err := s.notes.Create(ctx, note); err != nil {
		return nil, fmt.Errorf("failed to create note: %w", err)
	}
	return note, nil
}
