.PHONY: test test-cover test-verbose clean db-setup db-drop bench test-all help

# Database configuration for tests
DB_HOST ?= localhost
DB_PORT ?= 3306
DB_USER ?= root
DB_PASSWORD ?= root
DB_NAME ?= permission_test

# Colors for output
GREEN := \033[0;32m
RED := \033[0;31m
NC := \033[0m

# Run all tests
test:
	@echo "$(GREEN)Running tests with MariaDB...$(NC)"
	DB_HOST=$(DB_HOST) DB_PORT=$(DB_PORT) DB_USER=$(DB_USER) DB_PASSWORD=$(DB_PASSWORD) DB_NAME=$(DB_NAME) \
	go test -v ./tests

# Run tests with coverage
test-cover:
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	DB_HOST=$(DB_HOST) DB_PORT=$(DB_PORT) DB_USER=$(DB_USER) DB_PASSWORD=$(DB_PASSWORD) DB_NAME=$(DB_NAME) \
	go test -v ./tests -coverprofile=coverage.out
	go tool cover -html=coverage.out

# Run tests with verbose output
test-verbose:
	@echo "$(GREEN)Running tests with verbose output...$(NC)"
	DB_HOST=$(DB_HOST) DB_PORT=$(DB_PORT) DB_USER=$(DB_USER) DB_PASSWORD=$(DB_PASSWORD) DB_NAME=$(DB_NAME) \
	go test -v ./tests -v

# Clean test artifacts
clean:
	@echo "$(GREEN)Cleaning test artifacts...$(NC)"
	rm -f coverage.out
	go clean -testcache

# Setup test database
db-setup:
	@echo "$(GREEN)Setting up test database...$(NC)"
	mysql -h $(DB_HOST) -u $(DB_USER) -p$(DB_PASSWORD) -e "CREATE DATABASE IF NOT EXISTS $(DB_NAME);"
	mysql -h $(DB_HOST) -u $(DB_USER) -p$(DB_PASSWORD) $(DB_NAME) -e "DROP TABLE IF EXISTS role_user;"
	mysql -h $(DB_HOST) -u $(DB_USER) -p$(DB_PASSWORD) $(DB_NAME) -e "DROP TABLE IF EXISTS tenant_user;"
	mysql -h $(DB_HOST) -u $(DB_USER) -p$(DB_PASSWORD) $(DB_NAME) -e "DROP TABLE IF EXISTS role_permissions;"
	mysql -h $(DB_HOST) -u $(DB_USER) -p$(DB_PASSWORD) $(DB_NAME) -e "DROP TABLE IF EXISTS permissions;"
	mysql -h $(DB_HOST) -u $(DB_USER) -p$(DB_PASSWORD) $(DB_NAME) -e "DROP TABLE IF EXISTS roles;"
	mysql -h $(DB_HOST) -u $(DB_USER) -p$(DB_PASSWORD) $(DB_NAME) -e "DROP TABLE IF EXISTS tenants;"
	mysql -h $(DB_HOST) -u $(DB_USER) -p$(DB_PASSWORD) $(DB_NAME) -e "DROP TABLE IF EXISTS test_users;"
	@echo "$(GREEN)✅ Test database setup complete!$(NC)"

# Drop test database
db-drop:
	@echo "$(RED)Dropping test database...$(NC)"
	mysql -h $(DB_HOST) -u $(DB_USER) -p$(DB_PASSWORD) -e "DROP DATABASE IF EXISTS $(DB_NAME);"
	@echo "$(GREEN)✅ Test database dropped!$(NC)"

# Run benchmarks
bench:
	@echo "$(GREEN)Running benchmarks...$(NC)"
	DB_HOST=$(DB_HOST) DB_PORT=$(DB_PORT) DB_USER=$(DB_USER) DB_PASSWORD=$(DB_PASSWORD) DB_NAME=$(DB_NAME) \
	go test -v ./tests -bench=. -run=Benchmark

# Full test with setup
test-all: db-setup test

# Show help
help:
	@echo "Available targets:"
	@echo ""
	@echo "  $(GREEN)Testing:$(NC)"
	@echo "  make test          - Run all tests"
	@echo "  make test-verbose  - Run all tests with verbose output"
	@echo "  make test-cover    - Run tests with coverage report"
	@echo "  make test-all      - Setup DB and run all tests"
	@echo "  make bench         - Run benchmarks"
	@echo ""
	@echo "  $(GREEN)Database:$(NC)"
	@echo "  make db-setup      - Setup test database"
	@echo "  make db-drop       - Drop test database"
	@echo ""
	@echo "  $(GREEN)Maintenance:$(NC)"
	@echo "  make clean         - Clean test artifacts"
	@echo "  make help          - Show this help"
	@echo ""
	@echo "  $(GREEN)Configuration:$(NC)"
	@echo "  DB_HOST      - Database host (default: localhost)"
	@echo "  DB_PORT      - Database port (default: 3306)"
	@echo "  DB_USER      - Database user (default: root)"
	@echo "  DB_PASSWORD  - Database password (default: root)"
	@echo "  DB_NAME      - Database name (default: permission_test)"