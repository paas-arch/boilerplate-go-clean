// API сервиса. Панель запускает его из образа <репозиторий>/api и проверяет по /health.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	greetingsCreatePostHandler "app.internal/internal/api/handler/greetings_create_post"
	"app.internal/internal/config"
	greetingsCreatePostSchema "app.internal/internal/generated/api/schema/greetings_create_post"
	greeterService "app.internal/internal/service/greeter"
	createGreetingUsecase "app.internal/internal/usecase/create_greeting"
	"app.internal/internal/util/logger"
	"app.internal/internal/util/webserver"
)

// Подставляется при сборке: -ldflags "-X main.version=<тег образа>".
var version = "dev"

const (
	success = 0
	fail    = 1
)

func main() {
	os.Exit(run())
}

func run() (exitCode int) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log, err := logger.New()
	if err != nil {
		fmt.Println(err.Error())
		return fail
	}
	defer log.Flush()

	cfg, err := config.Load()
	if err != nil {
		log.Error(ctx, fmt.Errorf("failed to load config: %w", err))
		return fail
	}
	if err := log.EnableSentry(cfg.SentryDSN, cfg.Env); err != nil {
		log.Error(ctx, err)
		return fail
	}

	defer func() {
		if panicErr := recover(); panicErr != nil {
			log.Error(ctx, fmt.Errorf("recover panic: %v", panicErr))
			exitCode = fail
		}
	}()

	// База, clients и repositories — здесь, перед сервисами; образец подключения базы — _examples/postgres.

	// Service
	greeterSvc := greeterService.New()

	// Usecase
	createGreetingCase := createGreetingUsecase.New(greeterSvc)

	// Handler
	greetingsCreatePostMethod := greetingsCreatePostHandler.New(log, createGreetingCase)

	ws, err := webserver.NewServer(
		webserver.WithRecommendedSettings(webserver.RecommendedSettings{
			Host:              cfg.Host,
			Port:              config.Port,
			Logger:            log,
			TrustedProxyCIDRs: cfg.TrustedProxyCIDRs,
		}),
		webserver.WithHandler(greetingsCreatePostSchema.Handle(greetingsCreatePostMethod)),
	)
	if err != nil {
		log.Error(ctx, "failed to create server", err)
		return fail
	}

	go func() {
		log.Info(ctx, "api started", "version", version, "port", config.Port)
		if err := ws.ListenAndServe(); err != nil && !webserver.IsServerClosed(err) {
			log.Error(ctx, "api server error", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Info(ctx, "shutting down api")
	if err := ws.Shutdown(context.Background()); err != nil {
		log.Error(ctx, "failed to shutdown api", err)
		return fail
	}

	return success
}
