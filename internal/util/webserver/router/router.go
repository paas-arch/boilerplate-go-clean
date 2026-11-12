package router

import (
	"net/http"

	"app.internal/internal/util/webserver/response"
)

type (
	InternalErrHandler   func(err error, resp response.ResponseBuilder)
	BadRequestErrHandler func(err error, resp response.ResponseBuilder)
)

type ApiHandlerRouter interface {
	ApiHandler(method, path string, h http.HandlerFunc)
	URLParameters(r *http.Request) map[string]string
	IsStdMetricsEnabled() bool
}
