package request

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONBody_limitsBodySize(t *testing.T) {
	decode := func(valueBytes int) error {
		body := `{"value":"` + strings.Repeat("a", valueBytes) + `"}`
		req, err := NewRequest(httptest.NewRequest("POST", "/", strings.NewReader(body)))
		require.NoError(t, err)

		var dst struct{ Value string }
		return req.JSONBody(&dst)
	}

	require.NoError(t, decode(1024))
	require.Error(t, decode(maxBodyBytes))
}
