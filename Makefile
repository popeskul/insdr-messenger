DOCKER_COMPOSE = docker-compose
DB_USER = insider
DB_NAME = insider_db
DB_HOST = localhost
DB_PORT = 5432
DB_PASSWORD = password
DB_SSL = disable

DATABASE_URL ?= postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL)
DOCKER_DB_URL = postgres://$(DB_USER):$(DB_PASSWORD)@postgres:5432/$(DB_NAME)?sslmode=$(DB_SSL)

# Detect if Go is installed
GO_EXISTS := $(shell command -v go 2> /dev/null)

.PHONY: help
help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: info
info: ## Show project information and status
	@echo "======================================"
	@echo "   INSIDER MESSENGER"
	@echo "======================================"
	@echo ""
	@echo "📦 Project: Automatic Message Sending System"
	@echo "📁 Path: $(PWD)"
	@echo ""
	@echo "🔧 Environment:"
	@echo "   Go: $(if $(GO_EXISTS), installed $(shell go version | cut -d' ' -f3),✗ not installed - using Docker)"
	@echo "   Docker: $(shell docker --version 2>/dev/null | cut -d' ' -f3 || echo '✗ not installed')"
	@echo "   Docker Compose: $(shell docker-compose --version 2>/dev/null | cut -d' ' -f4 || echo '✗ not installed')"
	@echo ""
	@echo "🚀 Services Status:"
	@if docker-compose ps 2>/dev/null | grep -q "insider-messenger"; then \
		echo "   PostgreSQL: $$(docker-compose ps postgres 2>/dev/null | grep -q "healthy" && echo "✓ running (healthy)" || echo "⚠ running")"; \
		echo "   Redis: $$(docker-compose ps redis 2>/dev/null | grep -q "healthy" && echo "✓ running (healthy)" || echo "⚠ running")"; \
		echo "   App: $$(docker-compose ps app 2>/dev/null | grep -q "healthy" && echo "✓ running (healthy)" || echo "⚠ running")"; \
	else \
		echo "   Services: ✗ not running (run 'make up')"; \
	fi
	@echo ""
	@echo "🗄️  Database:"
	@echo "   Host: localhost:5432"
	@echo "   Name: insider_db"
	@echo "   User: insider"
	@if docker-compose ps 2>/dev/null | grep -q "postgres.*healthy"; then \
		echo "   Migration Status: $$(docker run --rm --network insider-messenger_default -v $(PWD)/migrations:/migrations migrate/migrate:v4.17.0 -path=/migrations -database="postgres://insider:password@postgres:5432/insider_db?sslmode=disable" version 2>/dev/null || echo "unknown")"; \
		echo "   Messages Count: $$(echo "SELECT count(*) FROM messages;" | docker-compose exec -T postgres psql -U insider -d insider_db -t 2>/dev/null || echo "unknown")"; \
	fi
	@echo ""
	@echo "📡 API Endpoints:"
	@echo "   Health: http://localhost:8080/health"
	@echo "   Messages: http://localhost:8080/api/v1/messages"
	@echo "   OpenAPI: http://localhost:8080/openapi"
	@echo ""
	@echo "⚙️  Configuration:"
	@echo "   Config File: config.docker.yaml"
	@echo "   Webhook URL: $$(grep -A1 "webhook:" config.docker.yaml 2>/dev/null | grep "url:" | awk '{print $$2}' || echo "not configured")"
	@echo "   Send Interval: 2 minutes"
	@echo ""
	@echo "📝 Quick Commands:"
	@echo "   Start: make dev"
	@echo "   Stop: make down"
	@echo "   Logs: make logs"
	@echo "   Tests: make test"
	@echo "   Help: make help"
	@echo ""
	@echo "======================================"

.PHONY: status
status: ## Show quick services status
	@echo "Services:"
	@docker-compose ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || echo "Services not running"

.PHONY: about
about: ## Show project description
	@echo ""
	@echo "🚀 INSIDER MESSENGER - Automatic Message Sending System"
	@echo ""
	@echo "A production-ready Go service that:"
	@echo "  • Sends messages automatically every 2 minutes"
	@echo "  • Stores messages in PostgreSQL"
	@echo "  • Caches sent messages in Redis"
	@echo "  • Provides RESTful API with OpenAPI spec"
	@echo "  • Includes health checks and monitoring"
	@echo ""
	@echo "Tech Stack:"
	@echo "  • Go 1.22+ with Chi router"
	@echo "  • PostgreSQL 15 for data storage"
	@echo "  • Redis 7 for caching"
	@echo "  • Docker & Docker Compose"
	@echo "  • Clean Architecture design"
	@echo ""

.PHONY: config
config: ## Show current configuration
	@echo "=== Current Configuration (config.docker.yaml) ==="
	@cat config.docker.yaml 2>/dev/null || echo "Configuration file not found"
	@echo ""
	@echo "=== Environment Variables ==="
	@echo "DATABASE_URL: $(DATABASE_URL)"
	@echo "DOCKER_DB_URL: $(DOCKER_DB_URL)"

# === DEVELOPMENT COMMANDS ===

.PHONY: up
up: ## Start all services (postgres, redis, app)
	@$(DOCKER_COMPOSE) up -d

.PHONY: down
down: ## Stop all services
	@$(DOCKER_COMPOSE) down

.PHONY: start
start: ## Start all services quietly
	@$(DOCKER_COMPOSE) up -d --wait > /dev/null 2>&1
	@docker run --rm --network insider-messenger_default \
		-v $(PWD)/migrations:/migrations \
		migrate/migrate:v4.17.0 \
		-path=/migrations \
		-database="postgres://insider:password@postgres:5432/insider_db?sslmode=disable" \
		up > /dev/null 2>&1 || true
	@echo "✅ Services started"

.PHONY: restart
restart: down up ## Restart all services

.PHONY: dev
dev: ## Start development (up + migrate) in detached mode
	@$(DOCKER_COMPOSE) up -d --wait
	@make migrate
	@echo ""
	@echo "✅ Development environment is running!"
	@echo ""
	@echo "📡 Services:"
	@echo "   - API: http://localhost:8080"
	@echo "   - Health: http://localhost:8080/health"
	@echo "   - PostgreSQL: localhost:5432"
	@echo "   - Redis: localhost:6379"
	@echo ""
	@echo "📝 Useful commands:"
	@echo "   - View logs: make logs"
	@echo "   - Check status: make status"
	@echo "   - Stop services: make down"
	@echo ""

.PHONY: dev-logs
dev-logs: ## Start development with logs attached
	@$(DOCKER_COMPOSE) up -d --wait
	@make migrate
	@make logs

.PHONY: reset
reset: ## Reset everything (remove containers and volumes)
	@echo "WARNING: This will delete all data!"
	@echo "Press Ctrl+C to cancel, or Enter to continue..."
	@read confirm
	@$(DOCKER_COMPOSE) down -v

.PHONY: destroy
destroy: ## Completely remove everything (containers, images, volumes, networks)
	@echo "⚠️  DANGER: This will completely remove:"
	@echo "   - All containers"
	@echo "   - All images" 
	@echo "   - All volumes (including database data)"
	@echo "   - All networks"
	@echo ""
	@echo "Type 'yes' to confirm: "
	@read confirm && [ "$$confirm" = "yes" ] || (echo "Cancelled" && exit 1)
	@echo "Removing containers and volumes..."
	@$(DOCKER_COMPOSE) down -v --remove-orphans || true
	@echo "Removing images..."
	@docker rmi insider-messenger-app:latest 2>/dev/null || true
	@docker rmi $$(docker images -q postgres:15-alpine) 2>/dev/null || true
	@docker rmi $$(docker images -q redis:7-alpine) 2>/dev/null || true
	@docker rmi $$(docker images -q migrate/migrate:v4.17.0) 2>/dev/null || true
	@echo "Cleaning up dangling images..."
	@docker image prune -f
	@echo "Removing unused networks..."
	@docker network prune -f
	@echo ""

.PHONY: clean-docker
clean-docker: ## Clean Docker resources (containers, volumes, images)
	@echo "Cleaning Docker resources..."
	@$(DOCKER_COMPOSE) down -v --remove-orphans 2>/dev/null || true
	@docker rmi insider-messenger-app:latest 2>/dev/null || true
	@echo "Docker resources cleaned"

.PHONY: logs
logs: ## Show app logs (usage: make logs [f=1] [n=50])
	@if [ "$${f}" = "1" ]; then \
		$(DOCKER_COMPOSE) logs -f app --tail $${n:-50}; \
	else \
		$(DOCKER_COMPOSE) logs app --tail $${n:-50}; \
	fi

.PHONY: logs-all
logs-all: ## Show logs from all services (usage: make logs-all [f=1] [n=50])
	@if [ "$${f}" = "1" ]; then \
		$(DOCKER_COMPOSE) logs -f --tail $${n:-50}; \
	else \
		$(DOCKER_COMPOSE) logs --tail $${n:-50}; \
	fi

.PHONY: rebuild
rebuild: down ## Rebuild and start services
	@$(DOCKER_COMPOSE) build
	@$(DOCKER_COMPOSE) up -d

# === MIGRATION COMMANDS ===

.PHONY: migrate
migrate: ## Run all pending migrations
	@echo "Running migrations..."
	@docker run --rm --network insider-messenger_default \
		-v $(PWD)/migrations:/migrations \
		migrate/migrate:v4.17.0 \
		-path=/migrations \
		-database="postgres://insider:password@postgres:5432/insider_db?sslmode=disable" \
		up

.PHONY: migrate-down
migrate-down:
	@echo "Rolling back last migration..."
	@docker run --rm --network insider-messenger_default \
		-v $(PWD)/migrations:/migrations \
		migrate/migrate:v4.17.0 \
		-path=/migrations \
		-database="postgres://insider:password@postgres:5432/insider_db?sslmode=disable" \
		down 1

.PHONY: migrate-new
migrate-new: ## Create new migration (usage: make migrate-new name=create_users)
	@if [ -z "$(name)" ]; then \
		echo "Error: Please provide migration name. Usage: make migrate-new name=your_migration_name"; \
		exit 1; \
	fi
	@echo "Creating migration: $(name)"
	@migrate create -ext sql -dir ./migrations -seq $(name)

.PHONY: migrate-status
migrate-status:
	@echo "Checking migration status..."
	@docker run --rm --network insider-messenger_default \
		-v $(PWD)/migrations:/migrations \
		migrate/migrate:v4.17.0 \
		-path=/migrations \
		-database="postgres://insider:password@postgres:5432/insider_db?sslmode=disable" \
		version

# === BUILD & TEST COMMANDS ===

.PHONY: build
build: ## Build binary
	@$(DOCKER_COMPOSE) build

.PHONY: test
test: ## Run tests
	@go test -v ./...

.PHONY: test-pkg
test-pkg: ## Run tests for specific package (usage: make test-pkg pkg=./internal/service)
	@if [ -z "$(pkg)" ]; then \
		echo "Error: Please specify package. Usage: make test-pkg pkg=./internal/service"; \
		exit 1; \
	fi
	@go test -v $(pkg)

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: run
run: ## Run app locally (requires Go)
	@go run cmd/server/main.go

# === SWAGGER AND API COMMANDS ===

# Include swagger commands from separate file
-include swagger.mk

.PHONY: generate
generate: ## Generate code from OpenAPI spec
	@echo "Installing oapi-codegen..."
	@go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	@echo "Generating code from OpenAPI spec..."
	@oapi-codegen -package api -generate types,chi-server -o internal/api/openapi.gen.go api/openapi.yaml
	@echo "✓ Code generated successfully"

.PHONY: swagger
swagger: swagger-setup ## Alias for swagger-setup

.PHONY: api-setup
api-setup: generate swagger ## Setup API with code generation and Swagger UI
	@echo "✓ API setup completed"

.PHONY: fmt
fmt:
	@go fmt ./...

.PHONY: lint
lint:
	@./.tools/golangci-lint run ./...

.PHONY: tidy
tidy:
	@go mod tidy

.PHONY: tidy-docker
tidy-docker: ## Tidy dependencies using Docker (no Go required)
	@docker run --rm -v $(PWD):/app -w /app golang:1.22-alpine go mod tidy

# === UTILITY COMMANDS ===

.PHONY: health
health:
	@curl -s http://localhost:8080/health | jq || echo "Application is not responding"

.PHONY: db-shell
db-shell: ## Open PostgreSQL shell
	@$(DOCKER_COMPOSE) exec postgres psql -U $(DB_USER) -d $(DB_NAME)

.PHONY: db-seed
db-seed: ## Seed database with test data
	@echo "Seeding database with test data..."
	@cat scripts/seed_data.sql | $(DOCKER_COMPOSE) exec -T postgres psql -U $(DB_USER) -d $(DB_NAME)

.PHONY: clean
clean: ## Clean build artifacts
	@rm -f insider-messenger
	@rm -f coverage.out

.PHONY: install-tools
install-tools: ## Install required tools (migrate, golangci-lint)
	@echo "Installing migrate tool..."
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.0 || echo "Failed to install migrate"
	@echo "Installing golangci-lint..."
	@if command -v brew >/dev/null 2>&1; then \
		brew install golangci-lint 2>/dev/null || brew upgrade golangci-lint 2>/dev/null || echo "golangci-lint might already be installed"; \
	else \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2 || echo "Failed to install golangci-lint"; \
	fi

# === SWAGGER COMMANDS ===

.PHONY: swagger-setup
swagger-setup: ## Download and setup Swagger UI
	@echo "Setting up Swagger UI..."
	@mkdir -p static/swagger-ui
	@if [ ! -f static/swagger-ui/swagger-ui.css ]; then \
		echo "Downloading Swagger UI assets..."; \
		curl -sL https://unpkg.com/swagger-ui-dist@5.10.3/swagger-ui.css -o static/swagger-ui/swagger-ui.css; \
		curl -sL https://unpkg.com/swagger-ui-dist@5.10.3/swagger-ui-bundle.js -o static/swagger-ui/swagger-ui-bundle.js; \
		curl -sL https://unpkg.com/swagger-ui-dist@5.10.3/swagger-ui-standalone-preset.js -o static/swagger-ui/swagger-ui-standalone-preset.js; \
		curl -sL https://unpkg.com/swagger-ui-dist@5.10.3/favicon-32x32.png -o static/swagger-ui/favicon-32x32.png; \
	fi
	@echo "✓ Swagger UI setup completed"
	@echo "Access at: http://localhost:8080/swagger/"

.PHONY: swagger-clean
swagger-clean: ## Remove Swagger UI files
	@rm -rf static/swagger-ui

.PHONY: swagger
swagger: swagger-setup ## Alias for swagger-setup

# === DOCUMENTATION COMMANDS ===

.PHONY: diagrams
diagrams: ## Generate PNG diagrams from PlantUML files
	@echo "Generating diagrams from PlantUML..."
	@if command -v plantuml >/dev/null 2>&1; then \
		plantuml -tpng docs/*.puml; \
		echo "✓ Diagrams generated in docs/"; \
	else \
		echo "PlantUML not installed. Using Docker..."; \
		docker run --rm -v $(PWD)/docs:/data plantuml/plantuml -tpng *.puml; \
	fi

.PHONY: diagrams-svg
diagrams-svg: ## Generate SVG diagrams from PlantUML files
	@echo "Generating SVG diagrams from PlantUML..."
	@if command -v plantuml >/dev/null 2>&1; then \
		plantuml -tsvg docs/*.puml; \
		echo "✓ SVG diagrams generated in docs/"; \
	else \
		echo "PlantUML not installed. Using Docker..."; \
		docker run --rm -v $(PWD)/docs:/data plantuml/plantuml -tsvg *.puml; \
	fi

.PHONY: docs
docs: diagrams ## Generate all documentation
	@echo "✓ Documentation generated"