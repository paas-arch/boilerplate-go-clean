package webserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"app.internal/internal/util/logger"
	"app.internal/internal/util/webserver/response"
	"app.internal/internal/util/webserver/router"
)

type HandlerFunc func(rt router.ApiHandlerRouter, internalErr router.InternalErrHandler, badReq router.BadRequestErrHandler)

// Handler is an alias for http.Handler, exported for use in WithPatternMW callbacks.
type Handler = http.Handler

// PatternMWFunc receives the route pattern and its handler; returns a (possibly wrapped) handler.
type PatternMWFunc func(pattern string, handler http.Handler) http.Handler

type RecommendedSettings struct {
	// Host пуст — слушаем все адреса.
	Host              string
	Port              int64
	Logger            logger.Log
	TrustedProxyCIDRs []string
}

type options struct {
	trustedProxyCIDRs []string
	host              string
	fallback          http.Handler
	port              int64
	logger            logger.Log
	handlers          []HandlerFunc
	patternMW         PatternMWFunc
	globalMW          []func(http.Handler) http.Handler
}

type Option func(*options)

func WithRecommendedSettings(s RecommendedSettings) Option {
	return func(o *options) {
		o.trustedProxyCIDRs = s.TrustedProxyCIDRs
		o.host = s.Host
		if s.Port != 0 {
			o.port = s.Port
		}
		if s.Logger != nil {
			o.logger = s.Logger
		}
	}
}

func WithHandler(fn HandlerFunc) Option {
	return func(o *options) {
		o.handlers = append(o.handlers, fn)
	}
}

// WithFallback отдаёт GET-запросы мимо /api/ этому обработчику: например, встроенную статику.
func WithFallback(h http.Handler) Option {
	return func(o *options) {
		o.fallback = h
	}
}

func WithPatternMW(fn PatternMWFunc) Option {
	return func(o *options) {
		o.patternMW = fn
	}
}

// WithGlobalMW подключает middleware, который оборачивает весь chi-роутер
// (выполняется до маршрутизации). Применяется к каждому запросу, включая 404.
func WithGlobalMW(mw func(http.Handler) http.Handler) Option {
	return func(o *options) {
		o.globalMW = append(o.globalMW, mw)
	}
}

type Server struct {
	srv *http.Server
}

func NewServer(opts ...Option) (*Server, error) {
	o := &options{port: 8080}
	for _, opt := range opts {
		opt(o)
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	if len(o.trustedProxyCIDRs) == 0 {
		r.Use(middleware.ClientIPFromRemoteAddr)
	} else {
		r.Use(middleware.ClientIPFromXFF(o.trustedProxyCIDRs...))
		// Цепочка только из доверенных адресов или битый заголовок адреса не дают: берём адрес соединения, иначе лимиты по IP получили бы пустую строку.
		r.Use(func(next http.Handler) http.Handler {
			fallback := middleware.ClientIPFromRemoteAddr(next)
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if middleware.GetClientIP(req.Context()) == "" {
					fallback.ServeHTTP(w, req)
					return
				}
				next.ServeHTTP(w, req)
			})
		})
	}
	// Свой recover вместо middleware.Recoverer: chi на панике отдаёт text/plain,
	// а нам нужен единый JSON-формат ошибок.
	r.Use(jsonRecoverer(o.logger))

	for _, mw := range o.globalMW {
		r.Use(mw)
	}

	internalErr := router.InternalErrHandler(func(err error, resp response.ResponseBuilder) {
		resp.SetCode(500).
			SetHeader("Content-Type", "application/json").
			SetBodyJsonData(map[string]string{"status": "error", "message": "internal server error"})
	})
	badReq := router.BadRequestErrHandler(func(err error, resp response.ResponseBuilder) {
		resp.SetCode(400).
			SetHeader("Content-Type", "application/json").
			SetBodyJsonData(map[string]string{"status": "error", "message": err.Error()})
	})

	// Неизвестный путь и неверный метод: chi по умолчанию отвечает text/plain.
	// Отдаём JSend-ошибку, чтобы весь API возвращал единый JSON-формат.
	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		if o.fallback != nil && req.Method == http.MethodGet && !strings.HasPrefix(req.URL.Path, "/api/") {
			o.fallback.ServeHTTP(w, req)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status":"error","message":"not found"}`))
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte(`{"status":"error","message":"method not allowed"}`))
	})

	// Встроенный health-check. Регистрируется напрямую на роутере, минуя
	// patternMW (auth), поэтому всегда публичен и не зависит от схемы/codegen.
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	cr := &chiRouter{
		r:         r,
		patternMW: o.patternMW,
	}
	for _, h := range o.handlers {
		h(cr, internalErr, badReq)
	}

	return &Server{
		srv: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", o.host, o.port),
			Handler:      r,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}, nil
}

func (s *Server) ListenAndServe() error              { return s.srv.ListenAndServe() }
func (s *Server) Shutdown(ctx context.Context) error { return s.srv.Shutdown(ctx) }
func IsServerClosed(err error) bool                  { return errors.Is(err, http.ErrServerClosed) }

type chiRouter struct {
	r         chi.Router
	patternMW PatternMWFunc
}

func (c *chiRouter) ApiHandler(method, path string, h http.HandlerFunc) {
	handler := http.Handler(h)
	if c.patternMW != nil {
		handler = c.patternMW(path, handler)
	}
	c.r.Method(method, path, handler)
}

func (c *chiRouter) URLParameters(r *http.Request) map[string]string {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return nil
	}
	params := make(map[string]string, len(rctx.URLParams.Keys))
	for i, k := range rctx.URLParams.Keys {
		params[k] = rctx.URLParams.Values[i]
	}
	return params
}

func (c *chiRouter) IsStdMetricsEnabled() bool { return false }

// jsonRecoverer перехватывает панику в хендлере, логирует её и отдаёт JSON 500
// в едином JSend-формате (chi middleware.Recoverer отвечает text/plain).
func jsonRecoverer(log logger.Log) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil && rec != http.ErrAbortHandler {
					if log != nil {
						log.Error(r.Context(), fmt.Errorf("panic recovered: %v", rec))
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"status":"error","message":"internal server error"}`))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
