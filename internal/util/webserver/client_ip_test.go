package webserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_ClientIP(t *testing.T) {
	tests := []struct {
		name, peer, xff, realIP, want string
		trusted                       []string
	}{
		{
			name: "прямое соединение",
			peer: "192.0.2.1:1234",
			want: "192.0.2.1",
		},
		{
			name:   "поддельные заголовки игнорируются",
			peer:   "192.0.2.1:1234",
			xff:    "203.0.113.9",
			realIP: "203.0.113.8",
			want:   "192.0.2.1",
		},
		{
			name:    "доверенный прокси",
			peer:    "10.0.0.1:1234",
			xff:     "192.0.2.1",
			trusted: []string{"10.0.0.0/24"},
			want:    "192.0.2.1",
		},
		{
			name:    "недоверенный прокси не меняет IP",
			peer:    "192.0.2.1:1234",
			xff:     "203.0.113.9",
			trusted: []string{"10.0.0.0/24"},
			want:    "192.0.2.1",
		},
		{
			name:    "поддельный левый адрес цепочки игнорируется",
			peer:    "10.0.0.1:1234",
			xff:     "203.0.113.9, 192.0.2.1, 10.0.0.2",
			trusted: []string{"10.0.0.0/24"},
			want:    "192.0.2.1",
		},
		{
			name:    "повреждённый заголовок доверенного прокси: адрес соединения",
			peer:    "10.0.0.1:1234",
			xff:     "bad",
			trusted: []string{"10.0.0.0/24"},
			want:    "10.0.0.1",
		},
		{
			name:    "в цепочке только доверенные адреса: адрес соединения",
			peer:    "10.0.0.3:1234",
			xff:     "10.0.0.1",
			trusted: []string{"10.0.0.0/24"},
			want:    "10.0.0.3",
		},
		{
			name:    "доверенный прокси без заголовка: адрес соединения",
			peer:    "10.0.0.3:1234",
			trusted: []string{"10.0.0.0/24"},
			want:    "10.0.0.3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			server, err := NewServer(WithRecommendedSettings(RecommendedSettings{TrustedProxyCIDRs: tt.trusted}), WithGlobalMW(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					got = middleware.GetClientIP(r.Context())
					next.ServeHTTP(w, r)
				})
			}))
			require.NoError(t, err)
			req := httptest.NewRequest("GET", "/health", nil)
			req.RemoteAddr = tt.peer
			req.Header.Set("X-Forwarded-For", tt.xff)
			req.Header.Set("X-Real-IP", tt.realIP)
			req.Header.Set("True-Client-IP", tt.realIP)
			server.srv.Handler.ServeHTTP(httptest.NewRecorder(), req)
			assert.Equal(t, tt.want, got)
		})
	}
}
