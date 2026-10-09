.PHONY: help build run test clean docker-up docker-down migrate-up migrate-down migrate-create swagger install-swag swagger-v2

# Load environment variables from .env (if present) and export to all commands
-include .env
export

# Tests run against PostgreSQL (internal/testutil/pgtest), each in its own
# throwaway schema. The URL is derived from the development database by swapping
# in a dedicated test database, so a test run never touches development data.
# Create it once with: createdb cosmo_agents_test
TEST_DATABASE_URL ?= $(subst /cosmo_agents?,/cosmo_agents_test?,$(SQLALCHEMY_POSTGRES_URI))
export TEST_DATABASE_URL

help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	@echo "Building cosmo-agents-go..."
	@go build -o bin/server ./cmd/server
	@go build -o bin/worker ./cmd/worker
	@echo "Build complete!"

run-server: ## Run the API server
	@echo "Starting server..."
	@go run ./cmd/server

run-worker: ## Run the background worker
	@echo "Starting worker..."
	@go run ./cmd/worker

run: ## Run both server and worker (requires separate terminals)
	@echo "Run 'make run-server' in one terminal and 'make run-worker' in another"

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -cover ./...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@go test -v -tags=integration ./tests/integration/...

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w .

tidy: ## Tidy go.mod
	@echo "Tidying go.mod..."
	@go mod tidy

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@echo "Clean complete!"

# Docker commands
docker-up: ## Start Docker services (Postgres, Redis)
	@echo "Starting Docker services..."
	@docker-compose -f docker/docker-compose.yml up -d
	@echo "Services started!"

docker-down: ## Stop Docker services
	@echo "Stopping Docker services..."
	@docker-compose -f docker/docker-compose.yml down
	@echo "Services stopped!"

docker-logs: ## View Docker logs
	@docker-compose -f docker/docker-compose.yml logs -f

# Migration commands
# Prefer a dedicated golang-migrate URL if provided; fallback to SQLALCHEMY_POSTGRES_URI
# Example: MIGRATE_DB_URL=postgres://postgres:postgres@localhost:5434/cosmo_backend_go?sslmode=disable
DB_URL ?= $(if $(MIGRATE_DB_URL),$(MIGRATE_DB_URL),$(SQLALCHEMY_POSTGRES_URI))

migrate-up: ## Run database migrations
	@echo "Running migrations..."
	@if [ -z "$(DB_URL)" ]; then echo "DB_URL is empty. Set MIGRATE_DB_URL=postgres://... or SQLALCHEMY_POSTGRES_URI"; exit 1; fi
	@migrate -path migrations -database "$(DB_URL)" up

migrate-down: ## Rollback last migration
	@echo "Rolling back migration..."
	@if [ -z "$(DB_URL)" ]; then echo "DB_URL is empty. Set MIGRATE_DB_URL=postgres://... or SQLALCHEMY_POSTGRES_URI"; exit 1; fi
	@migrate -path migrations -database "$(DB_URL)" down 1

migrate-create: ## Create a new migration (usage: make migrate-create name=your_migration_name)
	@echo "Creating migration: $(name)"
	@migrate create -ext sql -dir migrations -seq $(name)

migrate-force: ## Force migration version (usage: make migrate-force version=1)
	@echo "Forcing migration to version $(version)"
	@if [ -z "$(DB_URL)" ]; then echo "DB_URL is empty. Set MIGRATE_DB_URL=postgres://... or SQLALCHEMY_POSTGRES_URI"; exit 1; fi
	@migrate -path migrations -database "$(DB_URL)" force $(version)

bootstrap-migrations: ## Bootstrap 000001_init_schema from an existing DB (usage: make bootstrap-migrations BOOTSTRAP_DB_URL=postgres://...)
	@if [ -z "$(BOOTSTRAP_DB_URL)" ]; then \
		echo "Set BOOTSTRAP_DB_URL=postgres://user:pass@host:port/dbname?sslmode=disable"; \
		echo "Example: make bootstrap-migrations BOOTSTRAP_DB_URL=postgres://postgres:postgres@localhost:5434/cosmo_agents?sslmode=disable"; \
		exit 1; \
	fi
	@echo "Bootstrapping migrations from $(BOOTSTRAP_DB_URL) ..."
	@mkdir -p migrations
	@pg_dump --schema-only --no-owner --no-privileges "$(BOOTSTRAP_DB_URL)" > migrations/000001_init_schema.up.sql
	@pg_dump --schema-only --clean --if-exists --no-owner --no-privileges "$(BOOTSTRAP_DB_URL)" | sed -n '/^DROP /p' > migrations/000001_init_schema.down.sql
	@echo "Created migrations/000001_init_schema.up.sql and .down.sql"

# Development helpers
dev-setup: ## Setup development environment
	@echo "Setting up development environment..."
	@cp .env.example .env
	@echo "Created .env file. Please update with your configuration."
	@make docker-up
	@echo "Setup complete! Update .env and run 'make migrate-up' then 'make run-server'"

setup-s3: ## Setup MinIO S3 for Knowledge API
	@echo "Setting up S3 for Knowledge API..."
	@./scripts/setup-knowledge-s3.sh

install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "Tools installed!"

install-swag: ## Install swag CLI (swagger generator)
	@echo "Installing swag CLI..."
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "swag installed!"

swagger: ## Generate Swagger docs (scan repo)
	@echo "Generating Swagger docs..."
	@swag init -g cmd/server/main.go -o docs -d . --parseDependency --parseInternal --parseDepth 3
	@echo "Swagger docs generated at docs/"

swagger-v2: ## Generate Swagger docs for V2 only (if needed)
	@echo "Generating Swagger docs for V2..."
	@cd cmd/server && swag init -g main.go -o ../../docs --parseDependency --parseInternal
	@echo "Swagger docs (V2) generated at docs/"