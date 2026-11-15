// Package notebook — пример сервиса: прячет от сценариев, как заметки хранятся.
package notebook

import (
	"context"
	"errors"
	"fmt"

	"app.internal/internal/entity"
)

// latestLimit — сколько заметок отдаёт список: без предела ответ рос бы вместе с таблицей.
const latestLimit = 100

// ErrNoDatabase — базу сервису в панели не подключали: ручки отвечают об этом, а не внутренней ошибкой.
var ErrNoDatabase = errors.New("database is not connected")

type Service struct {
	notes noteRepository
}

// New без хранилища (nil) — сервис без базы: каждый метод возвращает ErrNoDatabase.
func New(notes noteRepository) *Service {
	return &Service{notes: notes}
}

func (s *Service) ListLatest(ctx context.Context) ([]*entity.Note, error) {
	if s.notes == nil {
		return nil, ErrNoDatabase
	}
	notes, err := s.notes.ListLatest(ctx, latestLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to list notes: %w", err)
	}
	return notes, nil
}

func (s *Service) Count(ctx context.Context) (int64, error) {
	if s.notes == nil {
		return 0, ErrNoDatabase
	}
	n, err := s.notes.Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count notes: %w", err)
	}
	return n, nil
}

func (s *Service) Add(ctx context.Context, text string) (*entity.Note, error) {
	if s.notes == nil {
		return nil, ErrNoDatabase
	}
	note := &entity.Note{Text: text}
	if err := s.notes.Create(ctx, note); err != nil {
		return nil, fmt.Errorf("failed to create note: %w", err)
	}
	return note, nil
}
