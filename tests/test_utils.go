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

// SetupTestDB sets up a test database using MariaDB
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
	db.Exec("DROP TABLE IF EXISTS role_user")
	db.Exec("DROP TABLE IF EXISTS tenant_user")
	db.Exec("DROP TABLE IF EXISTS role_permissions")
	db.Exec("DROP TABLE IF EXISTS permissions")
	db.Exec("DROP TABLE IF EXISTS roles")
	db.Exec("DROP TABLE IF EXISTS tenants")
	db.Exec("DROP TABLE IF EXISTS test_users")
	db.Exec("SET FOREIGN_KEY_CHECKS = 1")

	// Create the role_user table explicitly
	db.Exec(`CREATE TABLE IF NOT EXISTS role_user (
		user_id VARCHAR(100),
		role_id VARCHAR(100),
		PRIMARY KEY (user_id, role_id)
	)`)

	// Create the tenant_user table explicitly
	db.Exec(`CREATE TABLE IF NOT EXISTS tenant_user (
		user_id VARCHAR(100),
		tenant_id VARCHAR(100),
		PRIMARY KEY (user_id, tenant_id)
	)`)

	return db
}

// CleanupTestDB cleans up the test database
func CleanupTestDB(db *gorm.DB) {
	if db != nil {
		db.Exec("SET FOREIGN_KEY_CHECKS = 0")
		db.Exec("DROP TABLE IF EXISTS role_user")
		db.Exec("DROP TABLE IF EXISTS tenant_user")
		db.Exec("DROP TABLE IF EXISTS role_permissions")
		db.Exec("DROP TABLE IF EXISTS permissions")
		db.Exec("DROP TABLE IF EXISTS roles")
		db.Exec("DROP TABLE IF EXISTS tenants")
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
