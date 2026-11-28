// Образец воркера: раз в полминуты пишет в журнал, что жив. Скопируй в cmd/workers/<имя>/main.go и замени тело круга вызовом своего сервиса.
// Панель запускает воркер из образа <репозиторий>/workers/<имя>, если в paas.toml есть запись [[workers]] с этим name и enabled = true.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"app.internal/internal/config"
	"app.internal/internal/util/logger"
)

// Подставляется при сборке: -ldflags "-X main.version=<тег образа>".
var version = "dev"

// interval — как часто воркер просыпается; свой воркер выбирает его под свою работу.
const interval = 30 * time.Second

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

	// Зависимости — тем же порядком, что в cmd/api/main.go: база → clients → repositories → services.

	log.Info(ctx, "worker started", "version", version)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		// Ошибку круга пиши в журнал и иди дальше: упавший воркер панель перезапускает, и он выглядит сломанным.
		log.Info(ctx, "worker tick")

		// Остановка по сигналу: панель шлёт его при выкатке новой версии.
		select {
		case <-ctx.Done():
			log.Info(ctx, "worker stopped")
			return 0
		case <-ticker.C:
		}
	}
}
