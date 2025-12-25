.PHONY: help build run test clean docker-up docker-down migrate-up migrate-down

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	go build -o bin/url-shortener cmd/server/main.go

run: ## Run the application
	go run cmd/server/main.go

test: ## Run tests
	go test -v -cover ./...

test-coverage: ## Run tests with coverage report
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html

docker-up: ## Start all services (PostgreSQL, Redis, Prometheus)
	docker-compose up -d

docker-down: ## Stop all services
	docker-compose down

docker-logs: ## View docker logs
	docker-compose logs -f

migrate-up: ## Run database migrations
	docker exec -i url-shortener-postgres psql -U urlshortener -d urlshortener < migrations/001_initial_schema.sql

db-shell: ## Open PostgreSQL shell
	docker exec -it url-shortener-postgres psql -U urlshortener -d urlshortener

redis-shell: ## Open Redis shell
	docker exec -it url-shortener-redis redis-cli

lint: ## Run linter
	golangci-lint run

fmt: ## Format code
	go fmt ./...

deps: ## Download dependencies
	go mod download
	go mod tidy

dev: docker-up ## Start development environment
	@echo "Waiting for database to be ready..."
	@sleep 3
	@make migrate-up
	@echo "Starting application..."
	@make run
