package notes_get

import (
	"context"
	"fmt"

	"app.internal/internal/api/mapper"
	"app.internal/internal/generated/api/schema/components"
	schema "app.internal/internal/generated/api/schema/notes_get"
)

type Handler struct {
	log      logger
	notebook notebook
}

func New(log logger, notebook notebook) *Handler {
	return &Handler{log: log, notebook: notebook}
}

// Handle зовёт сервис напрямую: простое чтение без проверок и бизнес-ошибок сценария не требует.
func (h *Handler) Handle(ctx context.Context, _ schema.Parameters, resp schema.Response) error {
	notes, err := h.notebook.ListLatest(ctx)
	if err != nil {
		h.log.Error(ctx, fmt.Errorf("notesGet failed: %w", err))
		resp.InternalServerError(makeJSendError("Internal Server Error"))
		return err
	}

	resp.OK(mapper.MapNotes(notes))
	return nil
}

func makeJSendError(msg string) components.JSendError {
	return components.JSendError{
		Status:  components.EnumJSendErrorStatus_Error().Value(),
		Message: msg,
	}
}
