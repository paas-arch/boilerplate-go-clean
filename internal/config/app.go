// Package config читает настройки сервиса из окружения. Новая настройка — поле здесь и строка в [env_vars] файла paas.toml.
package config

import (
	"errors"
	"fmt"
	"net/netip"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

const envProduction = "production"

type Config struct {
	Env               string   `env:"APP_ENV" env-default:"local"`
	Host              string   `env:"APP_HOST"`
	SentryDSN         string   `env:"SENTRY_DSN"`
	TrustedProxyCIDRs []string `env:"TRUSTED_PROXY_CIDRS" env-separator:","`
}

const (
	// Port — порт API: панель ждёт сервис на нём и проверяет /health, поменять его нельзя.
	Port = 8080
)

func Load() (*Config, error) {
	var cfg Config

	if os.Getenv("APP_ENV") != envProduction {
		_ = godotenv.Load(".env") // В контейнере .env нет, настройки приходят из окружения.
	}
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("read env: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

// Validate проверяет то, что нельзя выразить тегами cleanenv.
func (c *Config) Validate() error {
	var errs []error

	for _, prefix := range c.TrustedProxyCIDRs {
		if _, err := netip.ParsePrefix(prefix); err != nil {
			errs = append(errs, errors.New("TRUSTED_PROXY_CIDRS must contain valid CIDRs"))
		}
	}

	return errors.Join(errs...)
}
