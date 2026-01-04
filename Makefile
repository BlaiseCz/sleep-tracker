.PHONY: help run build test test-unit test-int lint seed migrate docker-up docker-down docker-build clean swagger swagger-install

# Default target
help:
	@echo "Sleep Tracker - Available Commands"
	@echo ""
	@echo "Development:"
	@echo "  make run          - Start API server locally"
	@echo "  make build        - Build the binary"
	@echo "  make test         - Run all tests"
	@echo "  make test-unit    - Run unit tests only"
	@echo "  make test-int     - Run integration tests only"
	@echo "  make lint         - Run golangci-lint"
	@echo ""
	@echo "Database:"
	@echo "  make migrate      - Run database migrations"
	@echo "  make migrate-down - Rollback last migration"
	@echo "  make seed         - Load sample data"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-up    - Start all services (docker-compose up)"
	@echo "  make docker-down  - Stop all services (docker-compose down)"
	@echo "  make docker-build - Build Docker images"
	@echo "  make docker-dev   - Start development environment with hot reload"
	@echo ""
	@echo "Utilities:"
	@echo "  make clean        - Remove build artifacts"
	@echo "  make deps         - Download dependencies"

# =============================================================================
# Development
# =============================================================================

run:
	go run ./cmd/api

build:
	CGO_ENABLED=0 go build -ldflags="-w -s" -o bin/api ./cmd/api

test:
	go test -v -race -cover ./...

test-unit:
	go test -v -race -cover -short ./...

test-int:
	go test -v -race -cover -run Integration ./...

lint:
	golangci-lint run ./...

deps:
	go mod download
	go mod tidy

# =============================================================================
# Database
# =============================================================================

MIGRATE_CMD = migrate -path migrations -database "$(DATABASE_URL)"

migrate:
	$(MIGRATE_CMD) up

migrate-down:
	$(MIGRATE_CMD) down 1

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

seed:
	go run ./scripts/seed/main.go

# =============================================================================
# Docker
# =============================================================================

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-build:
	docker-compose build

docker-dev:
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml up

docker-logs:
	docker-compose logs -f

docker-clean:
	docker-compose down -v --rmi local

# =============================================================================
# Utilities
# =============================================================================

clean:
	rm -rf bin/ tmp/
	go clean -cache

# Generate Swagger documentation
swagger:
	swag init -g internal/api/handler/user_handler.go -o docs --parseDependency --parseInternal

swagger-install:
	go install github.com/swaggo/swag/cmd/swag@latest
