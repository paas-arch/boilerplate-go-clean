Правила — раздел «service» правила `.claude/rules/go-clean.md`. Образцы — `internal/service/greeter` (без хранения) и `_examples/postgres/internal/service/notebook` (над таблицей).

1. **deps.go** — только те методы клиентов, которые сервис вызывает (другие сервисы сюда не попадают):

```go
//go:generate echo $PWD - $GOPACKAGE - $GOFILE
//go:generate go tool go.uber.org/mock/mockgen -source=$GOFILE -destination=_mocks/$GOFILE -package=_mocks
package <mechanism>

type <name>Client interface {
	Get<Name>(ctx context.Context, name string) (*model.<Name>, error)
}
```

2. **service.go**:

```go
package <mechanism>

// Err<Name>NotFound — <почему исход выделен в sentinel>.
var Err<Name>NotFound = errors.New("<name> not found")

type Service struct {
	<name> <name>Client
}

func New(<name> <name>Client) *Service {
	return &Service{<name>: <name>}
}

func (s *Service) Get(ctx context.Context, name string) (*entity.<Name>, error) {
	x, err := s.<name>.Get<Name>(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get <name>: %w", err)
	}
	if x == nil {
		return nil, Err<Name>NotFound
	}
	return toEntity(x), nil
}
```

3. `go generate ./internal/service/<pkg>/...` — появился `_mocks/deps.go`.
4. **service_test.go** — табличный тест на каждый метод: успех, sentinel-исход, ошибка клиента. Готов, когда каждая ветка каждого публичного метода покрыта кейсом и `go test ./internal/service/<pkg>/...` зелёный.
5. Подключи в `main.go` в блоке `// Service`: `<camel>Svc := <camel>Service.New(<camel>Cl)`.
