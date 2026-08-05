package tests

import (
	"fmt"
	"os"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestUser represents a test user
type TestUser struct {
	ID    string `gorm:"primaryKey;type:varchar(100)"`
	Name  string `gorm:"type:varchar(255)"`
	Email string `gorm:"type:varchar(255);uniqueIndex"`
}

// TableName specifies the table name
func (TestUser) TableName() string {
	return "test_users"
}

// getEnv gets environment variable or returns default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// SetupTestDB sets up a test database using MariaDB (for standard testing)
func SetupTestDB(t *testing.T) *gorm.DB {
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "3306")
	dbUser := getEnv("DB_USER", "root")
	dbPassword := getEnv("DB_PASSWORD", "root")
	dbName := getEnv("DB_NAME", "permission_test")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser,
		dbPassword,
		dbHost,
		dbPort,
		dbName,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Clean up existing tables
	db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	db.Exec("DROP TABLE IF EXISTS model_has_permissions")
	db.Exec("DROP TABLE IF EXISTS model_has_roles")
	db.Exec("DROP TABLE IF EXISTS role_has_permissions")
	db.Exec("DROP TABLE IF EXISTS permissions")
	db.Exec("DROP TABLE IF EXISTS roles")
	db.Exec("DROP TABLE IF EXISTS test_users")
	db.Exec("SET FOREIGN_KEY_CHECKS = 1")

	// Create tables with new schema - FIXED UNIQUE INDEX
	createTables(db)

	return db
}

// createTables creates the necessary database tables
func createTables(db *gorm.DB) {
	// FIXED: Unique index includes tenant_id for tenant-specific permissions
	db.Exec(`CREATE TABLE IF NOT EXISTS permissions (
		id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		guard_name VARCHAR(100) DEFAULT 'web',
		tenant_id VARCHAR(100) NULL,
		created_at DATETIME NULL,
		updated_at DATETIME NULL,
		deleted_at DATETIME NULL,
		UNIQUE INDEX idx_permissions_name_guard_tenant (name, guard_name, tenant_id),
		INDEX idx_permissions_tenant (tenant_id)
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS roles (
		id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		guard_name VARCHAR(100) DEFAULT 'web',
		tenant_id VARCHAR(100) NULL,
		created_at DATETIME NULL,
		updated_at DATETIME NULL,
		deleted_at DATETIME NULL,
		UNIQUE INDEX idx_roles_name_guard_tenant (name, guard_name, tenant_id),
		INDEX idx_roles_tenant (tenant_id)
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS role_has_permissions (
		permission_id BIGINT UNSIGNED NOT NULL,
		role_id BIGINT UNSIGNED NOT NULL,
		tenant_id VARCHAR(100) NULL,
		PRIMARY KEY (permission_id, role_id),
		INDEX idx_role_permissions_tenant (tenant_id),
		FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE,
		FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS model_has_roles (
		role_id BIGINT UNSIGNED NOT NULL,
		model_type VARCHAR(255) NOT NULL,
		model_id BIGINT UNSIGNED NOT NULL,
		tenant_id VARCHAR(100) NULL,
		PRIMARY KEY (role_id, model_id, model_type),
		INDEX idx_model_has_roles_model (model_type, model_id),
		INDEX idx_model_roles_tenant (tenant_id),
		FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS model_has_permissions (
		permission_id BIGINT UNSIGNED NOT NULL,
		model_type VARCHAR(255) NOT NULL,
		model_id BIGINT UNSIGNED NOT NULL,
		tenant_id VARCHAR(100) NULL,
		PRIMARY KEY (permission_id, model_id, model_type),
		INDEX idx_model_has_permissions_model (model_type, model_id),
		INDEX idx_model_permissions_tenant (tenant_id),
		FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
	)`)
}

// CleanupTestDB cleans up the test database
func CleanupTestDB(db *gorm.DB) {
	if db != nil {
		db.Exec("SET FOREIGN_KEY_CHECKS = 0")
		db.Exec("DROP TABLE IF EXISTS model_has_permissions")
		db.Exec("DROP TABLE IF EXISTS model_has_roles")
		db.Exec("DROP TABLE IF EXISTS role_has_permissions")
		db.Exec("DROP TABLE IF EXISTS permissions")
		db.Exec("DROP TABLE IF EXISTS roles")
		db.Exec("DROP TABLE IF EXISTS test_users")
		db.Exec("SET FOREIGN_KEY_CHECKS = 1")

		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}
}

// CreateTestUser creates a test user
func CreateTestUser(db *gorm.DB, id, name, email string) error {
	user := &TestUser{
		ID:    id,
		Name:  name,
		Email: email,
	}
	return db.Create(user).Error
}

// NewTestDB creates a new test database (for Ginkgo tests)
func NewTestDB() (*gorm.DB, func()) {
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "3306")
	dbUser := getEnv("DB_USER", "root")
	dbPassword := getEnv("DB_PASSWORD", "root")
	dbName := getEnv("DB_NAME", "permission_test")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser,
		dbPassword,
		dbHost,
		dbPort,
		dbName,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to database: %v", err))
	}

	// Clean up existing tables
	db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	db.Exec("DROP TABLE IF EXISTS model_has_permissions")
	db.Exec("DROP TABLE IF EXISTS model_has_roles")
	db.Exec("DROP TABLE IF EXISTS role_has_permissions")
	db.Exec("DROP TABLE IF EXISTS permissions")
	db.Exec("DROP TABLE IF EXISTS roles")
	db.Exec("DROP TABLE IF EXISTS test_users")
	db.Exec("SET FOREIGN_KEY_CHECKS = 1")

	// Create tables
	createTables(db)

	cleanup := func() {
		db.Exec("SET FOREIGN_KEY_CHECKS = 0")
		db.Exec("DROP TABLE IF EXISTS model_has_permissions")
		db.Exec("DROP TABLE IF EXISTS model_has_roles")
		db.Exec("DROP TABLE IF EXISTS role_has_permissions")
		db.Exec("DROP TABLE IF EXISTS permissions")
		db.Exec("DROP TABLE IF EXISTS roles")
		db.Exec("DROP TABLE IF EXISTS test_users")
		db.Exec("SET FOREIGN_KEY_CHECKS = 1")

		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	return db, cleanup
}
