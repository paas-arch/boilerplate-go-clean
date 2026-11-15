package logger

import (
	"context"
	"errors"
	"testing"

	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogger_sendToSentry(t *testing.T) {
	tests := []struct {
		name          string
		sentryEnabled bool
		level         sentry.Level
		args          []interface{}
		wantEvents    int
		wantException string
		wantMessage   string
	}{
		{
			name:          "ошибка уходит исключением с полями контекста",
			sentryEnabled: true,
			level:         sentry.LevelError,
			args:          []interface{}{"failed to create server", errors.New("boom")},
			wantEvents:    1,
			wantException: "boom",
		},
		{
			name:          "warning без error уходит сообщением",
			sentryEnabled: true,
			level:         sentry.LevelWarning,
			args:          []interface{}{"slow query", "table", "notes"},
			wantEvents:    1,
			wantMessage:   "slow query",
		},
		{
			name:          "без DSN ничего не отправляется",
			sentryEnabled: false,
			level:         sentry.LevelError,
			args:          []interface{}{errors.New("boom")},
			wantEvents:    0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &sentry.MockTransport{}
			require.NoError(t, sentry.Init(sentry.ClientOptions{Dsn: "https://key@sentry.local/1", Transport: transport}))
			l := &Logger{z: zap.NewNop().Sugar(), sentry: tt.sentryEnabled}
			ctx := l.WithField(context.Background(), "user_id", 42)

			msg, _ := split(tt.args)
			l.sendToSentry(ctx, tt.level, msg, tt.args)

			events := transport.Events()
			require.Len(t, events, tt.wantEvents)
			if tt.wantEvents == 0 {
				return
			}
			e := events[0]
			assert.Equal(t, tt.level, e.Level)
			assert.Equal(t, 42, e.Contexts["log"]["user_id"])
			if tt.wantException != "" {
				require.NotEmpty(t, e.Exception)
				assert.Equal(t, tt.wantException, e.Exception[len(e.Exception)-1].Value)
			} else {
				assert.Equal(t, tt.wantMessage, e.Message)
			}
		})
	}
}

func TestSplit(t *testing.T) {
	tests := []struct {
		name    string
		args    []interface{}
		wantMsg string
		wantKV  []interface{}
	}{
		{
			name:    "сообщение и пары ключ значение",
			args:    []interface{}{"load wave", "clients", 1, "project", "cupfeed"},
			wantMsg: "load wave",
			wantKV:  []interface{}{"clients", 1, "project", "cupfeed"},
		},
		{
			name:    "ошибка после сообщения — поле error",
			args:    []interface{}{"failed to create server", errors.New("boom")},
			wantMsg: "failed to create server",
			wantKV:  []interface{}{"error", "boom"},
		},
		{
			name:    "одна ошибка — сообщение",
			args:    []interface{}{errors.New("boom")},
			wantMsg: "boom",
		},
		{
			name:    "значение без ключа — поле arg",
			args:    []interface{}{"server started", 8080},
			wantMsg: "server started",
			wantKV:  []interface{}{"arg", 8080},
		},
		{
			name:    "строка без пары — тоже arg",
			args:    []interface{}{"stopped", "by signal"},
			wantMsg: "stopped",
			wantKV:  []interface{}{"arg", "by signal"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, kv := split(tt.args)
			assert.Equal(t, tt.wantMsg, msg)
			assert.Equal(t, tt.wantKV, kv)
		})
	}
}

func TestLogger_Info_fields(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	l := &Logger{z: zap.New(core).Sugar()}

	l.Info(l.WithField(context.Background(), "request_id", "r1"), "load wave", "clients", 1)

	entries := logs.All()
	require.Len(t, entries, 1)
	assert.Equal(t, "load wave", entries[0].Message)
	assert.Equal(t, map[string]interface{}{"request_id": "r1", "clients": int64(1)}, entries[0].ContextMap())
}
