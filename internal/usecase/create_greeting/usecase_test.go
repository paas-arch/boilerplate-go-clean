package create_greeting

import (
	"context"
	"strings"
	"testing"

	"github.com/alekseyshavrak/errx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"app.internal/internal/entity"
	"app.internal/internal/usecase/create_greeting/_mocks"
)

func TestUsecase_Run(t *testing.T) {
	greeting := &entity.Greeting{Text: "Привет, Аня!"}

	tests := []struct {
		name      string
		input     string
		setup     func(g *_mocks.Mockgreeter)
		wantErrIs error
	}{
		{
			name:  "приветствие, пробелы по краям обрезаны",
			input: "  Аня  ",
			setup: func(g *_mocks.Mockgreeter) {
				g.EXPECT().Greet("Аня").Return(greeting)
			},
		},
		{name: "пустое имя", input: "   ", setup: func(*_mocks.Mockgreeter) {}, wantErrIs: errx.ErrValidation},
		{name: "слишком длинное имя", input: strings.Repeat("я", 101), setup: func(*_mocks.Mockgreeter) {}, wantErrIs: errx.ErrValidation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := _mocks.NewMockgreeter(gomock.NewController(t))
			tt.setup(g)

			got, err := New(g).Run(context.Background(), tt.input)

			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Same(t, greeting, got)
		})
	}
}
