# Local development. `make help` is the default.
# Pins match docs/features/01-project-foundation.md and CI.
#
# Host ports (docker-compose.yml): Postgres 5435→5432, Redis 6377→6379.

GOOSE_VERSION     := v3.24.3
SQLC_VERSION      := v1.29.0
GOLANGCI_VERSION  := v2.4.0
REDOCLY_VERSION   := 1.34.5

POSTGRES_HOST_PORT ?= 5435
DATABASE_URL       ?= postgres://friendfold:friendfold@127.0.0.1:$(POSTGRES_HOST_PORT)/friendfold?sslmode=disable

COMPOSE   ?= docker compose
GOTOOLCHAIN ?= auto

GOOSE := go run github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION)
SQLC  := go run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)

.DEFAULT_GOAL := help

.PHONY: help up down logs migrate migrate-down \
  api backend frontend dev \
  test test-backend test-frontend \
  lint lint-backend lint-frontend openapi \
  sqlc sqlc-diff typecheck

help: ## List targets
	@printf '%s\n' \
		'Friendfold local development. GNU make, macOS or Linux.' \
		'' \
		'Usage: make <target>' \
		'' \
		'Compose' \
		'  up              Start Postgres, Redis, Mailpit, MinIO (docker compose up -d)' \
		'  down            Stop Compose (keeps volumes)' \
		'  logs            Follow Compose logs' \
		'' \
		'Migrations' \
		'  migrate         goose $(GOOSE_VERSION) up (DSN host port $(POSTGRES_HOST_PORT))' \
		'  migrate-down    goose down (local/dev only)' \
		'' \
		'Apps' \
		'  api, backend    go run ./cmd/api (host :8080)' \
		'  frontend        pnpm install && pnpm dev (host :3000)' \
		'  dev             up + migrate, then api and frontend together' \
		'' \
		'Checks' \
		'  test            Backend + frontend tests' \
		'  test-backend    CGO_ENABLED=0 go test ./...' \
		'  test-frontend   frontend pnpm test' \
		'  lint            golangci-lint, ESLint, OpenAPI' \
		'  lint-backend    golangci-lint $(GOLANGCI_VERSION)' \
		'  lint-frontend   frontend pnpm lint' \
		'  openapi         Redocly CLI $(REDOCLY_VERSION)' \
		'  sqlc            sqlc $(SQLC_VERSION) generate' \
		'  sqlc-diff       sqlc diff (CI check)' \
		'  typecheck       frontend pnpm typecheck'

up: ## Start Compose services
	$(COMPOSE) up -d

down: ## Stop Compose services
	$(COMPOSE) down

logs: ## Follow Compose logs
	$(COMPOSE) logs -f

migrate: ## Apply goose migrations to Compose Postgres
	cd backend && GOTOOLCHAIN=$(GOTOOLCHAIN) $(GOOSE) -dir db/migrations postgres "$(DATABASE_URL)" up

migrate-down: ## Roll back the last goose migration
	cd backend && GOTOOLCHAIN=$(GOTOOLCHAIN) $(GOOSE) -dir db/migrations postgres "$(DATABASE_URL)" down

api: ## Run the Go API on the host
	cd backend && CGO_ENABLED=0 go run ./cmd/api

backend: api ## Alias for api

frontend: ## Install JS deps and run Next.js
	cd frontend && pnpm install && pnpm dev

dev: ## Compose + migrate + API and Next.js
	$(MAKE) up
	$(MAKE) migrate
	$(MAKE) -j2 api frontend

test: test-backend test-frontend ## Run all tests

test-backend: ## Go tests
	cd backend && CGO_ENABLED=0 go test ./...

test-frontend: ## Frontend unit tests
	cd frontend && pnpm install && pnpm test

lint: lint-backend lint-frontend openapi ## Lint Go, frontend, and OpenAPI

lint-backend: ## golangci-lint (CI pin)
	cd backend && GOTOOLCHAIN=$(GOTOOLCHAIN) $(GOLANGCI_LINT) run --config .golangci.yml ./...

lint-frontend: ## ESLint
	cd frontend && pnpm install && pnpm lint

openapi: ## Lint OpenAPI spec
	npx --yes @redocly/cli@$(REDOCLY_VERSION) lint --config redocly.yaml

sqlc: ## Generate sqlc code
	cd backend && GOTOOLCHAIN=$(GOTOOLCHAIN) $(SQLC) generate

sqlc-diff: ## Fail if generated sqlc code is stale
	cd backend && GOTOOLCHAIN=$(GOTOOLCHAIN) $(SQLC) diff

typecheck: ## Frontend typecheck
	cd frontend && pnpm install && pnpm typecheck
