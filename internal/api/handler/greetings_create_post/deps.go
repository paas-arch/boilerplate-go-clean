//go:generate echo $PWD - $GOPACKAGE - $GOFILE
//go:generate go tool go.uber.org/mock/mockgen -source=$GOFILE -destination=_mocks/$GOFILE -package=_mocks
package greetings_create_post

import (
	"context"

	"app.internal/internal/entity"
)

type logger interface {
	Error(context.Context, ...interface{})
}

type usecase interface {
	Run(ctx context.Context, name string) (*entity.Greeting, error)
}
