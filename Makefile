# 爱上杜美人智能开锁管理系统后端服务 Makefile
# =========================================

.PHONY: all build run test lint clean migrate seed reset-db docker-up docker-down help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Binary names
BINARY_NAME=smart-locker
BINARY_GATEWAY=api-gateway

# Directories
CMD_DIR=./cmd
INTERNAL_DIR=./internal
PKG_DIR=./pkg
MIGRATIONS_DIR=./migrations
SEEDS_DIR=./seeds
BUILD_DIR=./build

# Database
DB_HOST?=localhost
DB_PORT?=5432
DB_USER?=postgres
DB_PASSWORD?=postgres
DB_NAME?=smart_locker
DATABASE_URL?=postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

# Docker
DOCKER_COMPOSE=docker-compose -f deployments/docker/docker-compose.yml

# 默认目标
all: lint test build

# =========================================
# Build
# =========================================

build: ## Build the API gateway
	@echo "Building API Gateway..."
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_GATEWAY) $(CMD_DIR)/api-gateway

build-all: ## Build all services
	@echo "Building all services..."
	@for service in api-gateway user-service device-service order-service payment-service rental-service hotel-service mall-service distribution-service marketing-service finance-service content-service notification-service admin-service; do \
		echo "Building $$service..."; \
		$(GOBUILD) -o $(BUILD_DIR)/$$service $(CMD_DIR)/$$service || exit 1; \
	done
	@echo "All services built successfully!"

# =========================================
# Run
# =========================================

run: ## Run the API gateway
	@echo "Running API Gateway..."
	$(GOCMD) run $(CMD_DIR)/api-gateway/*.go

run-fast: build ## Build and run the API gateway (faster startup)
	@echo "Running API Gateway (pre-built)..."
	$(BUILD_DIR)/$(BINARY_GATEWAY)

run-watch: ## Run with hot-reload using air (install: go install github.com/air-verse/air@latest)
	@echo "Running API Gateway with hot-reload..."
	@command -v air >/dev/null 2>&1 || { echo "Installing air..."; go install github.com/air-verse/air@latest; }
	air

run-dev: docker-up ## Run in development mode with docker dependencies
	@echo "Running in development mode..."
	$(GOCMD) run $(CMD_DIR)/api-gateway/*.go

# =========================================
# Test
# =========================================

test: ## Run all tests
	@echo "Running tests..."
	$(GOTEST) -v -race ./...

test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	$(GOTEST) -v -race ./internal/... ./pkg/...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	$(GOTEST) -v -tags=integration ./tests/integration/...

test-e2e: ## Run end-to-end tests
	@echo "Running e2e tests..."
	$(GOTEST) -v -tags=e2e ./tests/e2e/...

test-api: ## Run API tests
	@echo "Running API tests..."
	$(GOTEST) -v -tags=api ./tests/api/...

coverage: ## Generate test coverage report
	@echo "Generating coverage report..."
	bash ./scripts/coverage.sh

coverage-gate: ## Verify coverage meets requirements
	@echo "Verifying coverage gate..."
	bash ./scripts/coverage-gate.sh

# =========================================
# Lint & Format
# =========================================

lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run ./...

lint-fix: ## Run linter with auto-fix
	@echo "Running linter with auto-fix..."
	golangci-lint run --fix ./...

fmt: ## Format code
	@echo "Formatting code..."
	$(GOFMT) ./...

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOVET) ./...

# =========================================
# Database
# =========================================

migrate: ## Run database migrations
	@echo "Running migrations..."
	./scripts/migrate.sh up

migrate-down: ## Rollback last migration
	@echo "Rolling back migration..."
	./scripts/migrate.sh down

migrate-reset: ## Reset all migrations
	@echo "Resetting migrations..."
	./scripts/migrate.sh reset

migrate-status: ## Show migration status
	@echo "Migration status..."
	./scripts/migrate.sh status

migrate-create: ## Create new migration (usage: make migrate-create name=create_xxx)
	@echo "Creating migration: $(name)..."
	./scripts/migrate.sh create $(name)

seed: ## Load seed data
	@echo "Loading seed data..."
	./scripts/seed.sh

reset-db: migrate-reset migrate seed ## Reset database and reload seed data
	@echo "Database reset completed!"

# =========================================
# Docker
# =========================================

docker-up: ## Start docker dependencies
	@echo "Starting docker dependencies..."
	$(DOCKER_COMPOSE) up -d

docker-down: ## Stop docker dependencies
	@echo "Stopping docker dependencies..."
	$(DOCKER_COMPOSE) down

docker-logs: ## Show docker logs
	$(DOCKER_COMPOSE) logs -f

docker-ps: ## Show docker container status
	$(DOCKER_COMPOSE) ps

docker-build: ## Build docker image for API gateway
	@echo "Building docker image..."
	docker build -f deployments/docker/Dockerfile -t $(BINARY_NAME):latest .

docker-push: ## Push docker image
	@echo "Pushing docker image..."
	docker push $(BINARY_NAME):latest

# =========================================
# Development
# =========================================

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download

deps-tidy: ## Tidy dependencies
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	$(GOGET) -u ./...
	$(GOMOD) tidy

generate: ## Run go generate
	@echo "Running go generate..."
	$(GOCMD) generate ./...

# Swagger CLI (installed via: go install github.com/swaggo/swag/cmd/swag@v1.16.6)
SWAG ?= $(shell go env GOPATH)/bin/swag

swagger: ## Generate swagger documentation
	@echo "Generating swagger documentation..."
	@mkdir -p .cache/go-build
	GOCACHE=$(CURDIR)/.cache/go-build $(SWAG) init --parseInternal --parseDependency --packagePrefix github.com/dumeirei/smart-locker-backend --dir ./cmd/api-gateway,./internal/handler -g main.go -o api/openapi

# =========================================
# Clean
# =========================================

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

clean-all: clean docker-down ## Clean all including docker
	@echo "All cleaned!"

# =========================================
# Help
# =========================================

help: ## Display this help
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
