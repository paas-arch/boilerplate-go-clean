package notes_create_post

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alekseyshavrak/errx"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"app.internal/internal/api/handler/notes_create_post/_mocks"
	"app.internal/internal/entity"
	"app.internal/internal/generated/api/schema/components"
	schema "app.internal/internal/generated/api/schema/notes_create_post"
	"app.internal/internal/util/webserver/response"
)

type stubRequest struct {
	schema.JsonRequest
	dto components.NoteCreateRequest
	err error
}

func (s stubRequest) DTO() (components.NoteCreateRequest, error) { return s.dto, s.err }

func TestHandler_Handle(t *testing.T) {
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		req      stubRequest
		setup    func(uc *_mocks.Mockusecase, log *_mocks.Mocklogger)
		wantCode int
		wantBody string
		wantErr  bool
	}{
		{
			name: "заметка добавлена",
			req:  stubRequest{dto: components.NoteCreateRequest{Text: "купить хлеб"}},
			setup: func(uc *_mocks.Mockusecase, _ *_mocks.Mocklogger) {
				uc.EXPECT().Run(gomock.Any(), "купить хлеб").Return(&entity.Note{ID: new(entity.NoteID(7)), Text: "купить хлеб", CreatedAt: &at}, nil)
			},
			wantCode: http.StatusOK,
			wantBody: `{"id":7,"text":"купить хлеб","createdAt":"2026-11-12T10:00:00Z"}`,
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
			req:  stubRequest{dto: components.NoteCreateRequest{Text: ""}},
			setup: func(uc *_mocks.Mockusecase, _ *_mocks.Mocklogger) {
				uc.EXPECT().Run(gomock.Any(), "").Return(nil, errx.Validation(errors.New("x"), errx.WithPublicMessage("Текст заметки: от 1 до 500 знаков")))
			},
			wantCode: http.StatusBadRequest,
			wantBody: `{"status":"error","message":"Текст заметки: от 1 до 500 знаков"}`,
		},
		{
			name: "база не подключена",
			req:  stubRequest{dto: components.NoteCreateRequest{Text: "купить хлеб"}},
			setup: func(uc *_mocks.Mockusecase, _ *_mocks.Mocklogger) {
				uc.EXPECT().Run(gomock.Any(), "купить хлеб").Return(nil, errx.Unavailable(errors.New("x"), errx.WithPublicMessage("База данных не подключена")))
			},
			wantCode: http.StatusServiceUnavailable,
			wantBody: `{"status":"error","message":"База данных не подключена"}`,
		},
		{
			name: "внутренняя ошибка",
			req:  stubRequest{dto: components.NoteCreateRequest{Text: "купить хлеб"}},
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
