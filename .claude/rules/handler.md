---
paths:
  - "internal/api/**"
---

# Слой handler

- Пакет `internal/api/handler/<operation_snake>`: `deps.go`, `handler.go`, `handler_test.go`, `_mocks/`. Имя пакета = имя сгенерированного пакета в `internal/generated/api/schema/`, импорт сгенерированного — с алиасом `schema`.
- HTTP-методы — только `GET` (чтение), `POST` (создание и изменение), `DELETE` (удаление).
- Параметры `POST` передаются только в JSON-теле (`requestBody`), не в query и не в path. Query — для `GET` и `DELETE`.
- Путь имеет вид `/api/v1/<группа>.<действие>`. Веб-приложение проекта пересылает бэкенду всё, что начинается с `/api/`, поэтому другие пути из браузера недоступны. Новую версию `/api/v2/...` добавляем при несовместимых изменениях контракта.
- `/health` встроен в `internal/util/webserver` и всегда отвечает 200: по нему панель понимает, что сервис запущен. Свою ручку на этот путь не заводим.
- Сигнатура `Handle` берётся из сгенерированного `handle.go`: `schema.JsonRequest` для тела, `schema.Parameters` для path/query.
- Порядок в `Handle`: разбор запроса (`req.DTO()` / `params.Path()` / `params.Query()`; ошибка → 400 "Invalid request") → вызов usecase → маппинг ошибок → ответ.
- Ошибки: `errx.ErrValidation` → 400, `ErrNotFound` → 404, `ErrForbidden` → 403, текст — `errx.PublicMessage(err)`. Остальное: `h.log.Error(ctx, fmt.Errorf("<op> failed: %w", err))`, 500 "Internal Server Error" и `return err`. Обработанная клиентская ошибка возвращает `nil`.
- Ответ — сгенерированные типы из `components`; ошибка — `makeJSendError(msg)` (локальная функция пакета, копируется как есть). Entity → DTO — функцией в `internal/api/mapper/<name>.go`.
- Middleware (авторизация, лимиты) навешивается в `main.go` через `webserver.WithPatternMW` по `<camel>Schema.HandlerPath`.
