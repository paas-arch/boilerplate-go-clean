Правила — раздел «client» правила `.claude/rules/go-clean.md`.

1. **Конфиг.** Поля в `internal/config` с тегами `env`/`env-default`, переменные — в `.env.example`, проверки обязательных значений — в `Config.Validate`. Если есть stub — константы `<X>Provider<Stub|Real>` и запрет stub в production.
2. **client.go**:

```go
package <provider>

const defaultBaseURL = "https://..."

type Config struct {
	APIKey  string
	Timeout time.Duration
	BaseURL string
}

type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

type APIError struct {
	Code int
	Text string
}

func (e *APIError) Error() string { return fmt.Sprintf("<provider> error %d: %s", e.Code, e.Text) }

func New(cfg Config) *Client {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{apiKey: cfg.APIKey, baseURL: baseURL, http: &http.Client{Timeout: cfg.Timeout}}
}
```

   Методы принимают `ctx`, возвращают типы пакета или `internal/model`, а не сырой JSON провайдера.
3. **stub/client.go** (если без ключей локально не обойтись): тот же набор методов, `log.Info(ctx, "<provider> stub", ...)`, успешный ответ.
4. **client_test.go** — подмена транспорта, сеть не трогаем:

```go
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func testClient(t *testing.T, response func(*http.Request) string) *Client {
	t.Helper()
	client := New(Config{APIKey: "secret", Timeout: time.Second})
	client.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": {"application/json"}},
			Body:       io.NopCloser(strings.NewReader(response(r))),
			Request:    r,
		}, nil
	})
	return client
}
```

   В `response` проверяй метод, путь и параметры запроса, возвращай JSON провайдера. Готов, когда каждый публичный метод покрыт успешным кейсом и `APIError`, тесты зелёные.
5. **main.go**, блок `// Client`: создание из `cfg`; при наличии stub — `switch cfg.<X>Provider { case stub: ...; case real: ...; default: return fail }`.
6. Добавь переменные в раздел «Configuration» `ARCHITECTURE.md`.

Потребитель (service или usecase) объявляет нужные методы клиента интерфейсом в своём `deps.go`.
