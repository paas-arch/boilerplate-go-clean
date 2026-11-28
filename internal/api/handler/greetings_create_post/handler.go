package greetings_create_post

import (
	"context"
	"errors"
	"fmt"

	"github.com/alekseyshavrak/errx"

	"app.internal/internal/api/mapper"
	"app.internal/internal/generated/api/schema/components"
	schema "app.internal/internal/generated/api/schema/greetings_create_post"
)

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

	greeting, err := h.uc.Run(ctx, dto.Name)
	if err != nil {
		if errors.Is(err, errx.ErrValidation) {
			resp.BadRequest(makeJSendError(errx.PublicMessage(err)))
			return nil
		}

		h.log.Error(ctx, fmt.Errorf("greetingsCreatePost failed: %w", err))
		resp.InternalServerError(makeJSendError("Internal Server Error"))
		return err
	}

	resp.OK(mapper.MapGreeting(greeting))
	return nil
}

func makeJSendError(msg string) components.JSendError {
	return components.JSendError{
		Status:  components.EnumJSendErrorStatus_Error().Value(),
		Message: msg,
	}
}
