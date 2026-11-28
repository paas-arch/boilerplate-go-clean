package psql

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPostgres_ErrorHidesPassword(t *testing.T) {
	const password = "s3cr3t-Pa55"

	tests := []struct {
		name string
		dsn  string
	}{
		{name: "база недоступна", dsn: "postgres://paas:" + password + "@127.0.0.1:1/paas?sslmode=disable"},
		{name: "адрес не разбирается", dsn: "postgres://paas:" + password + "@bad host/paas"},
		{name: "неизвестный параметр", dsn: "postgres://paas:" + password + "@127.0.0.1:1/paas?sslmode=nope"},
		{name: "формат ключ=значение", dsn: "host=127.0.0.1 port=1 user=paas password=" + password + " sslmode=nope"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			db, err := NewPostgres(ctx, tt.dsn)

			require.Error(t, err)
			assert.Nil(t, db)
			assert.NotContains(t, err.Error(), password)
			assert.Contains(t, err.Error(), "postgres connect")
		})
	}
}
