# .env нужен приложениям и goose; не экспортируем: приложения читают .env сами.
-include .env

.DEFAULT_GOAL := help

.PHONY: help
help: ## Список команд
	@grep -hE '^[a-zA-Z0-9_.-]+:.*## ' $(MAKEFILE_LIST) | awk -F ':.*## ' '{printf "  %-18s %s\n", $$1, $$2}'

.PHONY: run run.worker db.up db.down migrate.up migrate.create test lint codegen generate fmt
run: ## Запустить API (нужны .env и make db.up migrate.up)
	go run ./cmd/api
run.worker: ## Запустить воркер-пример. Другой: make run.worker name=<имя>
	go run ./cmd/workers/$(or $(name),example)
db.up: ## Поднять локальную базу и дождаться готовности
	docker compose up -d --wait db
db.down: ## Остановить локальную базу; данные остаются в томе
	docker compose down
migrate.up: ## Применить миграции из migrations/ к локальной базе
	go tool goose -dir migrations postgres "$(DATABASE_DSN)" up
migrate.create: ## Создать миграцию. Пример: make migrate.create name=create_orders
	@test -n "$(name)" || (echo "нужен name=<snake>" && exit 1)
	go tool goose -dir migrations -s create "$(name)" sql
test: ## Все тесты
	go test ./...
lint: ## Линтер
	golangci-lint run
codegen: ## Кодогенерация из api/schema.yaml
	@go run ./internal/util/webserver/codegen -spec api/schema.yaml -out internal/generated/api/schema -module app.internal
generate: ## Моки из deps.go
	go generate ./internal/...
fmt: ## Форматировать Go-код
	@go fmt ./...
