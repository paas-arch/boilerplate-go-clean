// Крон-пример: один раз пишет в журнал число заметок и завершается. Панель запускает его из образа <репозиторий>/cron/example
// по расписанию из записи [[crons]] в paas.toml. Свой крон — новая папка в cmd/cron/<имя> и запись с тем же name.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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

// run — один проход: крон делает работу и выходит, следующий запуск начнёт панель по расписанию. Код выхода не 0 — запуск упал.
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

	// Без DATABASE_DSN база в paas.toml не включена: крону считать нечего.
	notebook := notebookService.New(nil)
	if cfg.DatabaseDSN != "" {
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

	log.Info(ctx, "cron started", "version", version)
	n, err := notebook.Count(ctx)
	switch {
	case errors.Is(err, notebookService.ErrNoDatabase):
		log.Warning(ctx, "database is not connected: enable [postgresql] in paas.toml")
	case err != nil:
		log.Error(ctx, fmt.Errorf("failed to count notes: %w", err))
		return 1
	default:
		log.Info(ctx, "notes counted", "count", n)
	}
	return 0
}
