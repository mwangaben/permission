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

# Run specific test
test-specific:
	@echo "$(GREEN)Running specific test: $(TEST)...$(NC)"
	DB_HOST=$(DB_HOST) DB_PORT=$(DB_PORT) DB_USER=$(DB_USER) DB_PASSWORD=$(DB_PASSWORD) DB_NAME=$(DB_NAME) \
	go test -v ./tests -run $(TEST)

# Clean test artifacts
clean:
	@echo "$(GREEN)Cleaning test artifacts...$(NC)"
	rm -f coverage.out
	go clean -testcache

# Setup test database with new polymorphic schema
db-setup:
	@echo "$(GREEN)Setting up test database with polymorphic schema...$(NC)"
	mysql -h $(DB_HOST) -u $(DB_USER) -p$(DB_PASSWORD) -e "CREATE DATABASE IF NOT EXISTS $(DB_NAME);"
	mysql -h $(DB_HOST) -u $(DB_USER) -p$(DB_PASSWORD) $(DB_NAME) -e "DROP TABLE IF EXISTS model_has_permissions;"
	mysql -h $(DB_HOST) -u $(DB_USER) -p$(DB_PASSWORD) $(DB_NAME) -e "DROP TABLE IF EXISTS model_has_roles;"
	mysql -h $(DB_HOST) -u $(DB_USER) -p$(DB_PASSWORD) $(DB_NAME) -e "DROP TABLE IF EXISTS role_has_permissions;"
	mysql -h $(DB_HOST) -u $(DB_USER) -p$(DB_PASSWORD) $(DB_NAME) -e "DROP TABLE IF EXISTS permissions;"
	mysql -h $(DB_HOST) -u $(DB_USER) -p$(DB_PASSWORD) $(DB_NAME) -e "DROP TABLE IF EXISTS roles;"
	mysql -h $(DB_HOST) -u $(DB_USER) -p$(DB_PASSWORD) $(DB_NAME) -e "DROP TABLE IF EXISTS test_users;"
	@echo "$(GREEN)✅ Test database setup complete!$(NC)"
	@echo "$(GREEN)Created tables: permissions, roles, role_has_permissions, model_has_roles, model_has_permissions$(NC)"

# Create tables manually (if auto-migrate fails)
db-create-tables:
	@echo "$(GREEN)Creating tables manually...$(NC)"
	mysql -h $(DB_HOST) -u $(DB_USER) -p$(DB_PASSWORD) $(DB_NAME) -e "CREATE TABLE IF NOT EXISTS permissions (id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255) NOT NULL, guard_name VARCHAR(100) DEFAULT 'web', tenant_id VARCHAR(100) NULL, created_at DATETIME NULL, updated_at DATETIME NULL, deleted_at DATETIME NULL, UNIQUE INDEX idx_permissions_name_guard (name, guard_name), INDEX idx_permissions_tenant (tenant_id));"
	mysql -h $(DB_HOST) -u $(DB_USER) -p$(DB_PASSWORD) $(DB_NAME) -e "CREATE TABLE IF NOT EXISTS roles (id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255) NOT NULL, guard_name VARCHAR(100) DEFAULT 'web', tenant_id VARCHAR(100) NULL, created_at DATETIME NULL, updated_at DATETIME NULL, deleted_at DATETIME NULL, UNIQUE INDEX idx_roles_name_guard (name, guard_name), INDEX idx_roles_tenant (tenant_id));"
	mysql -h $(DB_HOST) -u $(DB_USER) -p$(DB_PASSWORD) $(DB_NAME) -e "CREATE TABLE IF NOT EXISTS role_has_permissions (permission_id BIGINT UNSIGNED NOT NULL, role_id BIGINT UNSIGNED NOT NULL, tenant_id VARCHAR(100) NULL, PRIMARY KEY (permission_id, role_id), INDEX idx_role_permissions_tenant (tenant_id), FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE, FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE);"
	mysql -h $(DB_HOST) -u $(DB_USER) -p$(DB_PASSWORD) $(DB_NAME) -e "CREATE TABLE IF NOT EXISTS model_has_roles (role_id BIGINT UNSIGNED NOT NULL, model_type VARCHAR(255) NOT NULL, model_id BIGINT UNSIGNED NOT NULL, tenant_id VARCHAR(100) NULL, PRIMARY KEY (role_id, model_id, model_type), INDEX idx_model_has_roles_model (model_type, model_id), INDEX idx_model_roles_tenant (tenant_id), FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE);"
	mysql -h $(DB_HOST) -u $(DB_USER) -p$(DB_PASSWORD) $(DB_NAME) -e "CREATE TABLE IF NOT EXISTS model_has_permissions (permission_id BIGINT UNSIGNED NOT NULL, model_type VARCHAR(255) NOT NULL, model_id BIGINT UNSIGNED NOT NULL, tenant_id VARCHAR(100) NULL, PRIMARY KEY (permission_id, model_id, model_type), INDEX idx_model_has_permissions_model (model_type, model_id), INDEX idx_model_permissions_tenant (tenant_id), FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE);"
	@echo "$(GREEN)✅ Tables created successfully$(NC)"

# Drop test database
db-drop:
	@echo "$(RED)Dropping test database...$(NC)"
	mysql -h $(DB_HOST) -u $(DB_USER) -p$(DB_PASSWORD) -e "DROP DATABASE IF EXISTS $(DB_NAME);"
	@echo "$(GREEN)✅ Test database dropped!$(NC)"

# Reset database (drop and recreate)
db-reset: db-drop db-setup

# Run benchmarks
bench:
	@echo "$(GREEN)Running benchmarks...$(NC)"
	DB_HOST=$(DB_HOST) DB_PORT=$(DB_PORT) DB_USER=$(DB_USER) DB_PASSWORD=$(DB_PASSWORD) DB_NAME=$(DB_NAME) \
	go test -v ./tests -bench=. -run=Benchmark

# Full test with setup
test-all: db-setup test

# Test with race detection
test-race:
	@echo "$(GREEN)Running tests with race detection...$(NC)"
	DB_HOST=$(DB_HOST) DB_PORT=$(DB_PORT) DB_USER=$(DB_USER) DB_PASSWORD=$(DB_PASSWORD) DB_NAME=$(DB_NAME) \
	go test -v ./tests -race

# Show help
help:
	@echo "Available targets:"
	@echo ""
	@echo "  $(GREEN)Testing:$(NC)"
	@echo "  make test              - Run all tests"
	@echo "  make test-verbose      - Run all tests with verbose output"
	@echo "  make test-cover        - Run tests with coverage report"
	@echo "  make test-race         - Run tests with race detection"
	@echo "  make test-specific     - Run specific test (usage: make test-specific TEST=TestName)"
	@echo "  make test-all          - Setup DB and run all tests"
	@echo "  make bench             - Run benchmarks"
	@echo ""
	@echo "  $(GREEN)Database:$(NC)"
	@echo "  make db-setup          - Setup test database (new polymorphic schema)"
	@echo "  make db-create-tables  - Create tables manually"
	@echo "  make db-drop           - Drop test database"
	@echo "  make db-reset          - Reset database (drop and recreate)"
	@echo ""
	@echo "  $(GREEN)Maintenance:$(NC)"
	@echo "  make clean             - Clean test artifacts"
	@echo "  make help              - Show this help"
	@echo ""
	@echo "  $(GREEN)Configuration:$(NC)"
	@echo "  DB_HOST      - Database host (default: localhost)"
	@echo "  DB_PORT      - Database port (default: 3306)"
	@echo "  DB_USER      - Database user (default: root)"
	@echo "  DB_PASSWORD  - Database password (default: root)"
	@echo "  DB_NAME      - Database name (default: permission_test)"
	@echo "  TEST         - Test name for test-specific (e.g., TestPermissionRegistration)"
	@echo ""
	@echo "  $(GREEN)Examples:$(NC)"
	@echo "  make test DB_PASSWORD=mysecret"
	@echo "  make test-all DB_USER=myuser DB_PASSWORD=mypass"
	@echo "  make test-specific TEST=TestPermissionRegistration"
	@echo "  make db-reset DB_NAME=my_permission_test"