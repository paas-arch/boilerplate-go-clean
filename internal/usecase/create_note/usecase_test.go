package create_note

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/alekseyshavrak/errx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"app.internal/internal/entity"
	"app.internal/internal/usecase/create_note/_mocks"
)

func TestUsecase_Run(t *testing.T) {
	note := &entity.Note{Text: "купить хлеб"}

	tests := []struct {
		name      string
		text      string
		setup     func(nb *_mocks.Mocknotebook)
		wantErrIs error
	}{
		{
			name: "заметка добавлена, пробелы по краям обрезаны",
			text: "  купить хлеб  ",
			setup: func(nb *_mocks.Mocknotebook) {
				nb.EXPECT().Add(gomock.Any(), "купить хлеб").Return(note, nil)
			},
		},
		{name: "пустой текст", text: "   ", setup: func(*_mocks.Mocknotebook) {}, wantErrIs: errx.ErrValidation},
		{name: "слишком длинный текст", text: strings.Repeat("я", 501), setup: func(*_mocks.Mocknotebook) {}, wantErrIs: errx.ErrValidation},
		{
			name: "ошибка базы",
			text: "купить хлеб",
			setup: func(nb *_mocks.Mocknotebook) {
				nb.EXPECT().Add(gomock.Any(), "купить хлеб").Return(nil, errors.New("db down"))
			},
			wantErrIs: errx.ErrInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nb := _mocks.NewMocknotebook(gomock.NewController(t))
			tt.setup(nb)

			got, err := New(nb).Run(context.Background(), tt.text)

			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Same(t, note, got)
		})
	}
}
