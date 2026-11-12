// API сервиса. Панель запускает его из образа <репозиторий>/api и проверяет по /health.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	trmsqlx "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"

	notesCreatePostHandler "app.internal/internal/api/handler/notes_create_post"
	notesGetHandler "app.internal/internal/api/handler/notes_get"
	"app.internal/internal/config"
	notesCreatePostSchema "app.internal/internal/generated/api/schema/notes_create_post"
	notesGetSchema "app.internal/internal/generated/api/schema/notes_get"
	noteRepository "app.internal/internal/repository/note"
	notebookService "app.internal/internal/service/notebook"
	createNoteUsecase "app.internal/internal/usecase/create_note"
	"app.internal/internal/util/logger"
	"app.internal/internal/util/psql"
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

	// Database: падаем на старте, если базы нет — так выкатка с неверным адресом видна как упавшая.
	dbCtx, dbCancel := context.WithTimeout(ctx, config.DatabaseConnectTimeout)
	db, err := psql.NewPostgres(dbCtx, cfg.DatabaseDSN)
	dbCancel()
	if err != nil {
		log.Error(ctx, fmt.Errorf("failed to connect to database: %w", err))
		return fail
	}
	defer func() { _ = db.Close() }()

	// Repository
	noteRepo := noteRepository.New(db, trmsqlx.DefaultCtxGetter)

	// Транзакции понадобятся сценарию, который пишет в несколько таблиц: tx.Do(ctx, func(ctx) error {...}).
	txManager, err := manager.New(trmsqlx.NewDefaultFactory(db))
	if err != nil {
		log.Error(ctx, fmt.Errorf("failed to init transaction manager: %w", err))
		return fail
	}
	_ = txManager

	// Service
	notebookSvc := notebookService.New(noteRepo)

	// Usecase
	createNoteCase := createNoteUsecase.New(notebookSvc)

	// Handler
	notesGetMethod := notesGetHandler.New(log, notebookSvc)
	notesCreatePostMethod := notesCreatePostHandler.New(log, createNoteCase)

	ws, err := webserver.NewServer(
		webserver.WithRecommendedSettings(webserver.RecommendedSettings{
			Host:              cfg.Host,
			Port:              cfg.Port,
			Logger:            log,
			TrustedProxyCIDRs: cfg.TrustedProxyCIDRs,
		}),
		webserver.WithHandler(notesGetSchema.Handle(notesGetMethod)),
		webserver.WithHandler(notesCreatePostSchema.Handle(notesCreatePostMethod)),
	)
	if err != nil {
		log.Error(ctx, "failed to create server", err)
		return fail
	}

	go func() {
		log.Info(ctx, "api started", "version", version, "port", cfg.Port)
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
