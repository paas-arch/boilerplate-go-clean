---
name: new-handler
description: Создание HTTP-ручки — от операции в api/schema.yaml через codegen до handler-пакета и маршрута в main.go. Используй при добавлении или изменении эндпоинта API.
---

Правила слоя — `.claude/rules/handler.md`.

1. **Контракт.** Добавь путь `/api/v1/<группа>.<действие>` (метод — только `GET`, `POST` или `DELETE`; у `POST` все параметры в JSON-теле, query и path не используются), `operationId` (camelCase, напр. `notesCreatePost`) и схемы в `api/schema.yaml`. Ответы ошибок — `JSendError`, успех без тела — `StatusResponse`. Описание полей — по-русски.
2. `make codegen` → появился `internal/generated/api/schema/<operation_snake>/`. Открой его `handle.go`: оттуда сигнатура `Handle`.
3. **deps.go**:

```go
//go:generate echo $PWD - $GOPACKAGE - $GOFILE
//go:generate go tool go.uber.org/mock/mockgen -source=$GOFILE -destination=_mocks/$GOFILE -package=_mocks
package <operation_snake>

type logger interface {
	Info(context.Context, ...interface{})
	Error(context.Context, ...interface{})
}

type usecase interface {
	Run(ctx context.Context, ...) (..., error)
}
```

4. **handler.go**:

```go
package <operation_snake>

import schema "<module>/internal/generated/api/schema/<operation_snake>"

type Handler struct {
	log logger
	uc  usecase
}

func New(log logger, uc usecase) *Handler {
	return &Handler{log: log, uc: uc}
}

func (h *Handler) Handle(ctx context.Context, req schema.JsonRequest, resp schema.Response) error {
	dto, err := req.DTO()
	if err != nil {
		resp.BadRequest(makeJSendError("Invalid request"))
		return nil
	}

	x, err := h.uc.Run(ctx, dto.Field)
	if err != nil {
		if errors.Is(err, errx.ErrValidation) {
			resp.BadRequest(makeJSendError(errx.PublicMessage(err)))
			return nil
		}
		if errors.Is(err, errx.ErrNotFound) {
			resp.NotFound(makeJSendError(errx.PublicMessage(err)))
			return nil
		}

		h.log.Error(ctx, fmt.Errorf("<operation> failed: %w", err))
		resp.InternalServerError(makeJSendError("Internal Server Error"))
		return err
	}

	resp.OK(mapper.<Name>Entity(x))
	return nil
}

func makeJSendError(msg string) components.JSendError {
	return components.JSendError{
		Status:  components.EnumJSendErrorStatus_Error().Value(),
		Message: msg,
	}
}
```

   Ветки `errors.Is` — ровно те, что возвращает usecase и что описаны ответами в схеме. Новая сущность в ответе — функция в `internal/api/mapper/<name>.go`.
5. `go generate ./internal/api/handler/<pkg>/...`.
6. **handler_test.go** — table-driven: `name`, вход, `setupMock func(uc *_mocks.Mockusecase)`, `setupLog` (только для 500), `wantCode`, `wantErr`. Запрос с телом — стаб `schema.JsonRequest`, с path/query — стаб `schema.Parameters` (тип с методами интерфейса, возвращающими заданные значения). Прогон:

```go
rb := response.NewHandlerResponse()
err := New(log, uc).Handle(context.Background(), tt.req, schema.NewResponse(rb))
rec := httptest.NewRecorder()
_ = rb.Send(rec)
assert.Equal(t, tt.wantCode, rec.Code)
```

   Готов, когда каждый HTTP-код из `Handle` покрыт кейсом и тесты зелёные.
7. **main.go** точки входа `cmd/api`: импорт handler и schema с алиасами `<camel>Handler`/`<camel>Schema`, `<camel>Method := <camel>Handler.New(log, <camel>Case)`, `webserver.WithHandler(<camel>Schema.Handle(<camel>Method))`.
8. Добавь строку в таблицу «API Endpoints» `ARCHITECTURE.md`.

Готов, когда `go build ./... && go test ./internal/api/...` зелёные и ручка видна в `ARCHITECTURE.md`.
