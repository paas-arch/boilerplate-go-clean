// Воркер-пример: раз в полминуты пишет в журнал число заметок. Панель запускает его из образа <репозиторий>/workers/example.
// Свой воркер — новая папка в cmd/workers/<имя>: имя из строчных латинских букв, цифр и дефиса.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	trmsqlx "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"

	"app.internal/internal/config"
	noteRepository "app.internal/internal/repository/note"
	notebookService "app.internal/internal/service/notebook"
	"app.internal/internal/util/logger"
	"app.internal/internal/util/psql"
)

// Подставляется при сборке: -ldflags "-X main.version=<тег образа>".
var version = "dev"

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log, err := logger.New()
	if err != nil {
		fmt.Println(err.Error())
		return 1
	}
	defer log.Flush()

	cfg, err := config.Load()
	if err != nil {
		log.Error(ctx, fmt.Errorf("failed to load config: %w", err))
		return 1
	}
	if err := log.EnableSentry(cfg.SentryDSN, cfg.Env); err != nil {
		log.Error(ctx, err)
		return 1
	}

	// Без DATABASE_DSN база в paas.toml не включена: воркер работает, а считать ему нечего.
	notebook := notebookService.New(nil)
	if cfg.DatabaseDSN == "" {
		// Один раз при старте: повтор на каждом круге выглядел бы в логах как сбой.
		log.Warning(ctx, "database is not connected: attach it on the service page in the panel")
	} else {
		dbCtx, dbCancel := context.WithTimeout(ctx, config.DatabaseConnectTimeout)
		db, err := psql.NewPostgres(dbCtx, cfg.DatabaseDSN)
		dbCancel()
		if err != nil {
			log.Error(ctx, fmt.Errorf("failed to connect to database: %w", err))
			return 1
		}
		defer func() { _ = db.Close() }()
		notebook = notebookService.New(noteRepository.New(db, trmsqlx.DefaultCtxGetter))
	}

	log.Info(ctx, "worker started", "version", version)
	ticker := time.NewTicker(config.ExampleWorkerInterval)
	defer ticker.Stop()
	for {
		n, err := notebook.Count(ctx)
		switch {
		case errors.Is(err, notebookService.ErrNoDatabase):
		case err != nil:
			log.Error(ctx, fmt.Errorf("failed to count notes: %w", err))
		default:
			log.Info(ctx, "notes counted", "count", n)
		}

		// Остановка по сигналу: панель шлёт его при выкатке новой версии.
		select {
		case <-ctx.Done():
			log.Info(ctx, "worker stopped")
			return 0
		case <-ticker.C:
		}
	}
}
