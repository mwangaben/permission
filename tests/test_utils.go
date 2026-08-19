package tests

import (
	"fmt"
	"github.com/mwangaben/factory/factory"
	"github.com/mwangaben/permission/models"
	"gorm.io/driver/postgres"
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
func SetupTestDB(t *testing.T) (*gorm.DB, func()) {
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

	err = factory.NewDatabaseHelper(db).RefreshDatabase(
		&models.Permission{},
		&models.Role{},
		&models.RoleHasPermission{},
		&models.ModelHasRole{},
		&models.ModelHasPermission{},
		&TestUser{},
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to refresh database: %v", err))
	}

	cleanup := func() {
		// Drop all tables using factory helper
		err := factory.NewDatabaseHelper(db).DropAllTables()
		if err != nil {
			fmt.Printf("Warning: Failed to cleanup tables: %v\n", err)
		}

		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}
	return db, cleanup
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

	err = factory.NewDatabaseHelper(db).RefreshDatabase(
		&models.Permission{},
		&models.Role{},
		&models.RoleHasPermission{},
		&models.ModelHasRole{},
		&models.ModelHasPermission{},
		&TestUser{},
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to refresh database: %v", err))
	}

	cleanup := func() {
		// Drop all tables using factory helper
		err := factory.NewDatabaseHelper(db).DropAllTables()
		if err != nil {
			fmt.Printf("Warning: Failed to cleanup tables: %v\n", err)
		}

		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	return db, cleanup
}

// NewTestDBPost creates a new test database (for Ginkgo tests)
func NewTestDBPost() (*gorm.DB, func()) {
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "benedictmwanga")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "permission_test")
	dbSSLMode := getEnv("DB_SSLMODE", "disable")

	// Build DSN for PostgreSQL
	var dsn string
	if dbPassword != "" {
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)
	} else {
		dsn = fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=%s",
			dbHost, dbPort, dbUser, dbName, dbSSLMode)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to database: %v", err))
	}

	err = factory.NewDatabaseHelper(db).RefreshDatabase(
		&models.Permission{},
		&models.Role{},
		&models.RoleHasPermission{},
		&models.ModelHasRole{},
		&models.ModelHasPermission{},
		&TestUser{},
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to refresh database: %v", err))
	}

	cleanup := func() {
		// Drop all tables using factory helper
		err := factory.NewDatabaseHelper(db).DropAllTables()
		if err != nil {
			fmt.Printf("Warning: Failed to cleanup tables: %v\n", err)
		}

		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	return db, cleanup
}
