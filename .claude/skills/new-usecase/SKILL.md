---
name: new-usecase
description: Создание usecase-пакета (один сценарий с методом Run, валидация ввода, перевод ошибок в errx) с deps.go, моками и тестами. Используй, когда ручке нужна бизнес-логика, бизнес-ошибка или оркестрация нескольких сервисов.
---

Правила слоя — `.claude/rules/usecase.md`.

1. **deps.go** — интерфейсы по ролям, только вызываемые методы:

```go
//go:generate echo $PWD - $GOPACKAGE - $GOFILE
//go:generate go tool go.uber.org/mock/mockgen -source=$GOFILE -destination=_mocks/$GOFILE -package=_mocks
package <verb_noun>

type notebook interface {
	Get(ctx context.Context, id entity.NoteID) (*entity.Note, error)
}
```

2. **usecase.go**:

```go
package <verb_noun>

type Usecase struct {
	notebook notebook
}

func New(notebook notebook) *Usecase {
	return &Usecase{notebook: notebook}
}

// Run <что делает и неочевидное «почему»>.
func (uc *Usecase) Run(ctx context.Context, id entity.NoteID) (*entity.Note, error) {
	// 1. Проверяем номер заметки
	if id <= 0 {
		return nil, errx.Validation(
			errors.New("invalid note id"),
			errx.WithPublicMessage("Не указана заметка"),
		)
	}

	// 2. Получаем заметку
	note, err := uc.notebook.Get(ctx, id)
	switch {
	case err == nil:
		return note, nil
	case errors.Is(err, notebook.ErrNoteNotFound):
		return nil, errx.NotFound(err, errx.WithPublicMessage("Заметка не найдена"))
	default:
		return nil, errx.Internal(err)
	}
}
```

3. `go generate ./internal/usecase/<pkg>/...`.
4. **usecase_test.go** — table-driven: `name`, вход, `setup func(mocks...)`, `wantErrIs` (`errx.ErrValidation`/`ErrNotFound`/`ErrForbidden`/`ErrInternal`). Кейс валидации проверяет, что в сервис не ходим (пустой `setup`). Готов, когда каждый `return` в `Run` покрыт кейсом и тесты зелёные.
5. Подключи в `main.go` в блоке `// Usecase`: `<camel>Case := <camel>Usecase.New(<camel>Svc)`.
