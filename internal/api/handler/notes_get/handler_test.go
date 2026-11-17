package notes_get

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"app.internal/internal/api/handler/notes_get/_mocks"
	"app.internal/internal/entity"
	schema "app.internal/internal/generated/api/schema/notes_get"
	notebookService "app.internal/internal/service/notebook"
	"app.internal/internal/util/webserver/response"
)

func TestHandler_Handle(t *testing.T) {
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		setup    func(nb *_mocks.Mocknotebook, log *_mocks.Mocklogger)
		wantCode int
		wantBody string
		wantErr  bool
	}{
		{
			name: "список заметок",
			setup: func(nb *_mocks.Mocknotebook, _ *_mocks.Mocklogger) {
				nb.EXPECT().ListLatest(gomock.Any()).Return([]*entity.Note{{ID: new(entity.NoteID(7)), Text: "купить хлеб", CreatedAt: &at}}, nil)
			},
			wantCode: http.StatusOK,
			wantBody: `{"notes":[{"id":7,"text":"купить хлеб","createdAt":"2026-11-12T10:00:00Z"}]}`,
		},
		{
			name: "заметок нет: пустой массив, а не null",
			setup: func(nb *_mocks.Mocknotebook, _ *_mocks.Mocklogger) {
				nb.EXPECT().ListLatest(gomock.Any()).Return(nil, nil)
			},
			wantCode: http.StatusOK,
			wantBody: `{"notes":[]}`,
		},
		{
			name: "внутренняя ошибка",
			setup: func(nb *_mocks.Mocknotebook, log *_mocks.Mocklogger) {
				nb.EXPECT().ListLatest(gomock.Any()).Return(nil, errors.New("db down"))
				log.EXPECT().Error(gomock.Any(), gomock.Any())
			},
			wantCode: http.StatusInternalServerError,
			wantBody: `{"status":"error","message":"Internal Server Error"}`,
			wantErr:  true,
		},
		{
			name: "база не подключена",
			setup: func(nb *_mocks.Mocknotebook, _ *_mocks.Mocklogger) {
				nb.EXPECT().ListLatest(gomock.Any()).Return(nil, notebookService.ErrNoDatabase)
			},
			wantCode: http.StatusServiceUnavailable,
			wantBody: `{"status":"error","message":"База данных не подключена: включите её в paas.toml (секция [postgresql]) и раскатите версию"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			nb := _mocks.NewMocknotebook(ctrl)
			log := _mocks.NewMocklogger(ctrl)
			tt.setup(nb, log)

			rb := response.NewHandlerResponse()
			err := New(log, nb).Handle(context.Background(), nil, schema.NewResponse(rb))
			rec := httptest.NewRecorder()
			_ = rb.Send(rec)

			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.wantCode, rec.Code)
			assert.JSONEq(t, tt.wantBody, rec.Body.String())
		})
	}
}
