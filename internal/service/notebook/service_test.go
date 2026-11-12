package notebook

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"app.internal/internal/entity"
	"app.internal/internal/service/notebook/_mocks"
)

func TestService_ListLatest(t *testing.T) {
	errDB := errors.New("db down")
	notes := []*entity.Note{{Text: "первая"}}

	tests := []struct {
		name    string
		notes   []*entity.Note
		repoErr error
	}{
		{name: "список с пределом", notes: notes},
		{name: "ошибка базы", repoErr: errDB},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := _mocks.NewMocknoteRepository(gomock.NewController(t))
			repo.EXPECT().ListLatest(gomock.Any(), uint64(latestLimit)).Return(tt.notes, tt.repoErr)

			got, err := New(repo).ListLatest(context.Background())

			assert.ErrorIs(t, err, tt.repoErr)
			assert.Equal(t, tt.notes, got)
		})
	}
}

func TestService_Add(t *testing.T) {
	errDB := errors.New("db down")

	tests := []struct {
		name    string
		repoErr error
	}{
		{name: "заметка добавлена"},
		{name: "ошибка базы", repoErr: errDB},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := _mocks.NewMocknoteRepository(gomock.NewController(t))
			repo.EXPECT().Create(gomock.Any(), &entity.Note{Text: "текст"}).Return(tt.repoErr)

			got, err := New(repo).Add(context.Background(), "текст")

			if tt.repoErr != nil {
				assert.ErrorIs(t, err, tt.repoErr)
				assert.Nil(t, got)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, &entity.Note{Text: "текст"}, got)
		})
	}
}

func TestService_Count(t *testing.T) {
	errDB := errors.New("db down")

	tests := []struct {
		name    string
		n       int64
		repoErr error
	}{
		{name: "число заметок", n: 3},
		{name: "ошибка базы", repoErr: errDB},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := _mocks.NewMocknoteRepository(gomock.NewController(t))
			repo.EXPECT().Count(gomock.Any()).Return(tt.n, tt.repoErr)

			got, err := New(repo).Count(context.Background())

			assert.ErrorIs(t, err, tt.repoErr)
			assert.Equal(t, tt.n, got)
		})
	}
}
