//go:generate echo $PWD - $GOPACKAGE - $GOFILE
//go:generate go tool go.uber.org/mock/mockgen -source=$GOFILE -destination=_mocks/$GOFILE -package=_mocks
package notebook

import (
	"context"

	"app.internal/internal/entity"
)

type noteRepository interface {
	ListLatest(ctx context.Context, limit uint64) ([]*entity.Note, error)
	Count(ctx context.Context) (int64, error)
	Create(ctx context.Context, note *entity.Note) error
}
