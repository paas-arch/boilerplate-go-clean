package greeter

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"app.internal/internal/entity"
)

func TestService_Greet(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  *entity.Greeting
	}{
		{name: "приветствие по имени", input: "Аня", want: &entity.Greeting{Text: "Привет, Аня!"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, New().Greet(tt.input))
		})
	}
}
