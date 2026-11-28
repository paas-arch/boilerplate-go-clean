// Package greeter — пример сервиса: доменная логика без хранения. Сервис над таблицей — _examples/postgres/internal/service/notebook.
package greeter

import "app.internal/internal/entity"

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) Greet(name string) *entity.Greeting {
	return &entity.Greeting{Text: "Привет, " + name + "!"}
}
