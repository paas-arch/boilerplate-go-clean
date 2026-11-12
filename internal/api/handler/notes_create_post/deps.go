//go:generate echo $PWD - $GOPACKAGE - $GOFILE
//go:generate go tool go.uber.org/mock/mockgen -source=$GOFILE -destination=_mocks/$GOFILE -package=_mocks
package notes_create_post

import (
	"context"

	"app.internal/internal/entity"
)

type logger interface {
	Error(context.Context, ...interface{})
}

type usecase interface {
	Run(ctx context.Context, text string) (*entity.Note, error)
}
