package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/lib/pq"

	"github.com/mwangaben/factory/factory"
	"github.com/mwangaben/permission/models"
	"github.com/mwangaben/permission/storage"
	"github.com/mwangaben/permission/storage/entstore"
	"github.com/mwangaben/permission/storage/entstore/ent"
	"github.com/mwangaben/permission/storage/gormstore"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestUser represents a test user (a polymorphic target for roles/permissions).
type TestUser struct {
	ID    string `gorm:"primaryKey;type:varchar(100)"`
	Name  string `gorm:"type:varchar(255)"`
	Email string `gorm:"type:varchar(255);uniqueIndex"`
}

func (TestUser) TableName() string {
	return "test_users"
}

// ─── Environment helpers ────────────────────────────────────────────────

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// sharedGormModels lists every GORM-managed table for the permission system,
// plus the test user table. Used by both GORM setup paths (MariaDB and
// Postgres) to keep them in sync.
func sharedGormModels() []interface{} {
	return []interface{}{
		&models.Permission{},
		&models.Role{},
		&models.RoleHasPermission{},
		&models.ModelHasRole{},
		&models.ModelHasPermission{},
		&models.Tenant{},
		&models.TenantUser{},
		&TestUser{},
	}
}

// ─── GORM-backed test databases ─────────────────────────────────────────

// SetupTestDB sets up a test database using MariaDB.
func SetupTestDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()

	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "3306")
	dbUser := getEnv("DB_USER", "root")
	dbPassword := getEnv("DB_PASSWORD", "root")
	dbName := getEnv("DB_NAME", "permission_test")

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=5s",
		dbUser, dbPassword, dbHost, dbPort, dbName,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to MariaDB: %v", err)
	}

	pingDB(t, db)

	if err := factory.NewDatabaseHelper(db).RefreshDatabase(sharedGormModels()...); err != nil {
		t.Fatalf("Failed to refresh database: %v", err)
	}

	return db, gormCleanup(db)
}

// NewTestDB is the testing.T-free variant used by Ginkgo's BeforeEach.
func NewTestDB() (*gorm.DB, func()) {
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "3306")
	dbUser := getEnv("DB_USER", "root")
	dbPassword := getEnv("DB_PASSWORD", "root")
	dbName := getEnv("DB_NAME", "permission_test")

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=5s",
		dbUser, dbPassword, dbHost, dbPort, dbName,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to MariaDB: %v", err))
	}

	if err := factory.NewDatabaseHelper(db).RefreshDatabase(sharedGormModels()...); err != nil {
		panic(fmt.Sprintf("Failed to refresh database: %v", err))
	}

	return db, gormCleanup(db)
}

// NewTestDBPost creates a Postgres-backed test database (for Ginkgo).
func NewTestDBPost() (*gorm.DB, func()) {
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "benedictmwanga")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "permission_test")
	dbSSLMode := getEnv("DB_SSLMODE", "disable")

	var dsn string
	if dbPassword != "" {
		dsn = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s connect_timeout=5",
			dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode,
		)
	} else {
		dsn = fmt.Sprintf(
			"host=%s port=%s user=%s dbname=%s sslmode=%s connect_timeout=5",
			dbHost, dbPort, dbUser, dbName, dbSSLMode,
		)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to Postgres: %v", err))
	}

	if err := factory.NewDatabaseHelper(db).RefreshDatabase(sharedGormModels()...); err != nil {
		panic(fmt.Sprintf("Failed to refresh database: %v", err))
	}

	return db, gormCleanup(db)
}

// gormCleanup drops all tables and closes the connection.
func gormCleanup(db *gorm.DB) func() {
	return func() {
		if err := factory.NewDatabaseHelper(db).DropAllTables(); err != nil {
			fmt.Printf("Warning: Failed to cleanup tables: %v\n", err)
		}
		if sqlDB, err := db.DB(); err == nil && sqlDB != nil {
			sqlDB.Close()
		}
	}
}

// pingDB verifies connectivity within a short deadline so tests fail fast.
func pingDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get sql.DB: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		t.Fatalf("Database unreachable: %v", err)
	}
}

// ─── Ent-backed test database ───────────────────────────────────────────

// NewTestDBEnt sets up an Ent client against Postgres, dropping all tables
// and letting Ent recreate its schema.
//
// The test_users table is managed by GORM because Ent doesn't own it; we
// migrate it separately via a short-lived GORM connection.
func NewTestDBEnt() (*ent.Client, func()) {
	dsn := buildPostgresDSN()

	drv, err := entsql.Open(dialect.Postgres, dsn)
	if err != nil {
		panic(fmt.Sprintf("Failed to open Ent driver: %v", err))
	}

	client := ent.NewClient(ent.Driver(drv))

	// Ensure test_users exists via GORM (drop + migrate only that table).
	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to open GORM for test_users: %v", err))
	}
	if err := gdb.Migrator().DropTable(&TestUser{}); err != nil {
		// ignore; table may not exist
	}
	if err := gdb.AutoMigrate(&TestUser{}); err != nil {
		panic(fmt.Sprintf("Failed to migrate test_users: %v", err))
	}
	if sqlDB, err := gdb.DB(); err == nil && sqlDB != nil {
		sqlDB.Close()
	}

	// Drop Ent-managed tables and let Ent recreate them with its own schema.
	// This ensures both backends can run against the same physical DB
	// without column-type drift between runs.
	dropEntTables(client)

	if err := client.Schema.Create(context.Background()); err != nil {
		panic(fmt.Sprintf("Failed to create Ent schema: %v", err))
	}

	cleanup := func() {
		dropEntTables(client)
		_ = client.Close()
	}
	return client, cleanup
}

func dropEntTables(client *ent.Client) {
	ctx := context.Background()
	// Order is defensive; no FKs currently.
	_, _ = client.RoleHasPermission.Delete().Exec(ctx)
	_, _ = client.ModelHasRole.Delete().Exec(ctx)
	_, _ = client.ModelHasPermission.Delete().Exec(ctx)
	_, _ = client.Permission.Delete().Exec(ctx)
	_, _ = client.Role.Delete().Exec(ctx)
	_, _ = client.TenantUser.Delete().Exec(ctx)
	_, _ = client.Tenant.Delete().Exec(ctx)
}

func buildPostgresDSN() string {
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "benedictmwanga")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "permission_test")
	dbSSLMode := getEnv("DB_SSLMODE", "disable")

	if dbPassword != "" {
		return fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s connect_timeout=5",
			dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode,
		)
	}
	return fmt.Sprintf(
		"host=%s port=%s user=%s dbname=%s sslmode=%s connect_timeout=5",
		dbHost, dbPort, dbUser, dbName, dbSSLMode,
	)
}

// ─── Repository factories (for backend-parameterized tests) ─────────────

// NewGormRepository builds a storage.Repository against a fresh Postgres DB.
// Returns the repo, the raw DB (for test_users setup / assertions), and a
// cleanup func.
func NewGormRepository() (storage.Repository, *gorm.DB, func()) {
	db, cleanup := NewTestDBPost()

	repo, err := gormstore.New(db)
	if err != nil {
		panic(fmt.Sprintf("Failed to build GORM repository: %v", err))
	}
	return repo, db, cleanup
}

// NewEntRepository builds a storage.Repository against a fresh Ent DB.
// Returns the repo and a cleanup func.
//
// Note: to also set up test_users for a specific test, call
// NewTestDBPost() separately or use the GORM handle from a GORM test.
func NewEntRepository() (storage.Repository, func()) {
	client, cleanup := NewTestDBEnt()

	repo, err := entstore.New(client)
	if err != nil {
		panic(fmt.Sprintf("Failed to build Ent repository: %v", err))
	}
	return repo, cleanup
}

// ─── User helper ────────────────────────────────────────────────────────

func CreateTestUser(db *gorm.DB, id, name, email string) error {
	return db.Create(&TestUser{ID: id, Name: name, Email: email}).Error
}
