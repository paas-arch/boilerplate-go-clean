// Package create_note — пример сценария: проверка ввода и перевод ошибок в errx.
package create_note

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/alekseyshavrak/errx"

	"app.internal/internal/entity"
	notebookService "app.internal/internal/service/notebook"
)

// maxTextRunes — предел длины заметки: длинный текст — уже документ, а не заметка.
const maxTextRunes = 500

type Usecase struct {
	notebook notebook
}

func New(notebook notebook) *Usecase {
	return &Usecase{notebook: notebook}
}

func (uc *Usecase) Run(ctx context.Context, text string) (*entity.Note, error) {
	// 1. Проверяем текст, не обращаясь к базе
	text = strings.TrimSpace(text)
	if text == "" || utf8.RuneCountInString(text) > maxTextRunes {
		return nil, errx.Validation(errors.New("invalid note text"), errx.WithPublicMessage("Текст заметки: от 1 до 500 знаков"))
	}

	// 2. Сохраняем заметку
	note, err := uc.notebook.Add(ctx, text)
	if errors.Is(err, notebookService.ErrNoDatabase) {
		return nil, errx.Unavailable(err, errx.WithPublicMessage("База данных не подключена: включите её в paas.toml (секция [postgresql]) и раскатите версию"))
	}
	if err != nil {
		return nil, errx.Internal(err)
	}

	return note, nil
}
