// Package psql подключает сервис к Postgres; миграции применяет отдельный образ migrator.
package psql

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// NewPostgres открывает пул и проверяет соединение: sqlx.ConnectContext сам делает ping.
func NewPostgres(ctx context.Context, dsn string) (*sqlx.DB, error) {
	db, err := sqlx.ConnectContext(ctx, "postgres", dsn)
	if err != nil {
		if db != nil {
			_ = db.Close()
		}
		return nil, fmt.Errorf("postgres connect: %w", redact(err, dsn))
	}

	return db, nil
}

var (
	urlCredentials  = regexp.MustCompile(`://[^@/\s"]*@`)
	keywordPassword = regexp.MustCompile(`password=('[^']*'|\S+)`)
)

// redact убирает из текста ошибки адрес базы и пароль: разбор кривого адреса возвращает его целиком.
// Цепочку ошибок не сохраняем нарочно: через неё исходный текст с паролем дошёл бы до лога.
func redact(err error, dsn string) error {
	text := strings.ReplaceAll(err.Error(), dsn, "<dsn>")
	text = urlCredentials.ReplaceAllString(text, "://<credentials>@")
	text = keywordPassword.ReplaceAllString(text, "password=<password>")

	return errors.New(text)
}
