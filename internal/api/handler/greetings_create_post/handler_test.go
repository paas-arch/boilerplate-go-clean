package greetings_create_post

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alekseyshavrak/errx"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"app.internal/internal/api/handler/greetings_create_post/_mocks"
	"app.internal/internal/entity"
	"app.internal/internal/generated/api/schema/components"
	schema "app.internal/internal/generated/api/schema/greetings_create_post"
	"app.internal/internal/util/webserver/response"
)

type stubRequest struct {
	schema.JsonRequest
	dto components.GreetingCreateRequest
	err error
}

func (s stubRequest) DTO() (components.GreetingCreateRequest, error) { return s.dto, s.err }

func TestHandler_Handle(t *testing.T) {
	tests := []struct {
		name     string
		req      stubRequest
		setup    func(uc *_mocks.Mockusecase, log *_mocks.Mocklogger)
		wantCode int
		wantBody string
		wantErr  bool
	}{
		{
			name: "приветствие",
			req:  stubRequest{dto: components.GreetingCreateRequest{Name: "Аня"}},
			setup: func(uc *_mocks.Mockusecase, _ *_mocks.Mocklogger) {
				uc.EXPECT().Run(gomock.Any(), "Аня").Return(&entity.Greeting{Text: "Привет, Аня!"}, nil)
			},
			wantCode: http.StatusOK,
			wantBody: `{"text":"Привет, Аня!"}`,
		},
		{
			name:     "тело не разобрано",
			req:      stubRequest{err: errors.New("bad json")},
			setup:    func(*_mocks.Mockusecase, *_mocks.Mocklogger) {},
			wantCode: http.StatusBadRequest,
			wantBody: `{"status":"error","message":"Invalid request"}`,
		},
		{
			name: "ошибка проверки",
			req:  stubRequest{dto: components.GreetingCreateRequest{Name: ""}},
			setup: func(uc *_mocks.Mockusecase, _ *_mocks.Mocklogger) {
				uc.EXPECT().Run(gomock.Any(), "").Return(nil, errx.Validation(errors.New("x"), errx.WithPublicMessage("Имя: от 1 до 100 знаков")))
			},
			wantCode: http.StatusBadRequest,
			wantBody: `{"status":"error","message":"Имя: от 1 до 100 знаков"}`,
		},
		{
			name: "внутренняя ошибка",
			req:  stubRequest{dto: components.GreetingCreateRequest{Name: "Аня"}},
			setup: func(uc *_mocks.Mockusecase, log *_mocks.Mocklogger) {
				uc.EXPECT().Run(gomock.Any(), gomock.Any()).Return(nil, errx.Internal(errors.New("boom")))
				log.EXPECT().Error(gomock.Any(), gomock.Any())
			},
			wantCode: http.StatusInternalServerError,
			wantBody: `{"status":"error","message":"Internal Server Error"}`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := _mocks.NewMockusecase(ctrl)
			log := _mocks.NewMocklogger(ctrl)
			tt.setup(uc, log)

			rb := response.NewHandlerResponse()
			err := New(log, uc).Handle(context.Background(), tt.req, schema.NewResponse(rb))
			rec := httptest.NewRecorder()
			_ = rb.Send(rec)

			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.wantCode, rec.Code)
			assert.JSONEq(t, tt.wantBody, rec.Body.String())
		})
	}
}
