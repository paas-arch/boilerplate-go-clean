# .env нужен приложениям; не экспортируем: приложения читают .env сами.
-include .env

.DEFAULT_GOAL := help

.PHONY: help
help: ## Список команд
	@grep -hE '^[a-zA-Z0-9_.-]+:.*## ' $(MAKEFILE_LIST) | awk -F ':.*## ' '{printf "  %-18s %s\n", $$1, $$2}'

.PHONY: run run.worker run.cron check test lint codegen generate fmt
run: ## Запустить API (нужен .env: cp .env.example .env)
	go run ./cmd/api
run.worker: ## Запустить воркер: make run.worker name=<имя>
	@test -n "$(name)" || (echo "нужен name=<имя>" && exit 1)
	go run ./cmd/workers/$(name)
run.cron: ## Один запуск крона: make run.cron name=<имя>
	@test -n "$(name)" || (echo "нужен name=<имя>" && exit 1)
	go run ./cmd/cron/$(name)
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
# Имя репозитория — из origin: paas-check сверяет его с name в paas.toml, как сборка.
check: ## Всё, что проверят CI и сборка: формат, codegen и моки, линтер, тесты, paas.toml
	@test -z "$$(gofmt -l .)" || { echo "не отформатированы, запусти make fmt:"; gofmt -l .; exit 1; }
	$(MAKE) codegen generate
	golangci-lint run
	go test ./...
	docker run --rm -v "$(CURDIR):/src" ghcr.io/paas-arch/paas-check:1 -repo "$$(basename -s .git "$$(git remote get-url origin)")"
