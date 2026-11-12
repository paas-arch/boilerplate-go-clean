//go:generate echo $PWD - $GOPACKAGE - $GOFILE
//go:generate go tool go.uber.org/mock/mockgen -source=$GOFILE -destination=_mocks/$GOFILE -package=_mocks
package create_note

import (
	"context"

	"app.internal/internal/entity"
)

type notebook interface {
	Add(ctx context.Context, text string) (*entity.Note, error)
}
