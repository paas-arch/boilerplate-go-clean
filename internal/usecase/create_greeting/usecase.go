// Package create_greeting — пример сценария: проверка ввода и перевод ошибок в errx.
package create_greeting

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/alekseyshavrak/errx"

	"app.internal/internal/entity"
)

// maxNameRunes — предел длины имени: длиннее — уже не имя, а текст.
const maxNameRunes = 100

type Usecase struct {
	greeter greeter
}

func New(greeter greeter) *Usecase {
	return &Usecase{greeter: greeter}
}

func (uc *Usecase) Run(_ context.Context, name string) (*entity.Greeting, error) {
	// 1. Проверяем имя
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxNameRunes {
		return nil, errx.Validation(errors.New("invalid name"), errx.WithPublicMessage("Имя: от 1 до 100 знаков"))
	}

	// 2. Здороваемся
	return uc.greeter.Greet(name), nil
}
