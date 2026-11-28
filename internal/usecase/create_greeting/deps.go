//go:generate echo $PWD - $GOPACKAGE - $GOFILE
//go:generate go tool go.uber.org/mock/mockgen -source=$GOFILE -destination=_mocks/$GOFILE -package=_mocks
package create_greeting

import "app.internal/internal/entity"

type greeter interface {
	Greet(name string) *entity.Greeting
}
