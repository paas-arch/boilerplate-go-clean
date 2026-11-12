---
paths:
  - "internal/client/**"
---

# Слой client

- Пакет `internal/client/<provider>`: обёртка над одним внешним API. Своей бизнес-логики нет — только транспорт, сериализация и перевод ответов провайдера в типы пакета или `internal/model`.
- `type Config struct` + `New(cfg Config) *Client`; если конфиг может оказаться неполным — `(*Client, error)` с проверкой в `New`. Значения в Config приходят из `internal/config` через `main.go`, сам клиент env не читает.
- HTTP — `net/http` с таймаутом из конфига; секреты не попадают в логи и ошибки.
- Ошибки провайдера — типизированные (`type APIError struct{Code int; Text string}` + хелпер `IsCode`), чтобы сервис или usecase мог ветвиться.
- Локальная разработка без ключей — `internal/client/<provider>/stub`: тот же метод, пишет в лог и отвечает успехом. Выбор stub/реальный — по переменной `<X>_PROVIDER` в `main.go`; stub запрещён в production через `config.Validate`.
- Ключи провайдера задаются переменными окружения сервиса; на сервере их кладут в секреты сервиса в панели.
- Тест клиента — `client_test.go` с подменой `http.Transport` (каркас в скиле `new-client`): проверяем запрос и разбор ответа, сеть не трогаем.
