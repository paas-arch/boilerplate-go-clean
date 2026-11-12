package logger

import (
	"context"
	"errors"
	"testing"

	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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
			args:          []interface{}{"slow ", "query"},
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

			l.sendToSentry(ctx, tt.level, tt.args)

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
