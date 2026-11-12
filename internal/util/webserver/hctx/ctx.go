package hctx

import (
	"context"
	"net/http"
)

func MakeRequestContext(r *http.Request) context.Context {
	return r.Context()
}
