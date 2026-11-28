package mapper

import (
	"app.internal/internal/entity"
	"app.internal/internal/generated/api/schema/components"
)

func MapGreeting(g *entity.Greeting) components.Greeting {
	return components.Greeting{Text: g.Text}
}
