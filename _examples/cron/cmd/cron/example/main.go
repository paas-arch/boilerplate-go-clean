// Образец крона: один проход — запись в журнал — и выход. Скопируй в cmd/cron/<имя>/main.go и замени проход вызовом своего сервиса.
// Панель запускает крон из образа <репозиторий>/cron/<имя> по расписанию из записи [[crons]] с этим name в paas.toml.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"app.internal/internal/config"
	"app.internal/internal/util/logger"
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

	// Зависимости — тем же порядком, что в cmd/api/main.go: база → clients → repositories → services.

	log.Info(ctx, "cron started", "version", version)
	// Проход: вызов сервиса. Ошибка — log.Error и return 1, чтобы панель показала запуск упавшим.
	log.Info(ctx, "cron done")
	return 0
}
