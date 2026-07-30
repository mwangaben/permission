package tests

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/mwangaben/permission/models"
	"github.com/mwangaben/permission/permission"
	"github.com/mwangaben/permission/role"
)

func TestPermissionRegistration(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(db)

	db.AutoMigrate(&models.Permission{})

	permReg := permission.NewRegistrar(db)

	perm := &models.Permission{
		ID:          uuid.New().String(),
		Name:        "user.view",
		Model:       "user",
		Action:      "view",
		DisplayName: "View Users",
		Description: "Can view user list and user details",
	}

	t.Run("Register permission", func(t *testing.T) {
		err := permReg.Register(perm)
		if err != nil {
			t.Fatalf("Failed to register permission: %v", err)
		}

		var found models.Permission
		if err := db.Where("name = ?", "user.view").First(&found).Error; err != nil {
			t.Fatalf("Permission not found: %v", err)
		}

		if found.Name != "user.view" {
			t.Errorf("Expected name 'user.view', got '%s'", found.Name)
		}
		if found.Model != "user" {
			t.Errorf("Expected model 'user', got '%s'", found.Model)
		}
		if found.Action != "view" {
			t.Errorf("Expected action 'view', got '%s'", found.Action)
		}
	})

	t.Run("Register duplicate permission (should update)", func(t *testing.T) {
		perm := &models.Permission{
			Name:        "user.view",
			DisplayName: "View Users Updated",
			Description: "Updated description",
			Model:       "user",
			Action:      "view",
		}

		err := permReg.Register(perm)
		if err != nil {
			t.Fatalf("Failed to register duplicate permission: %v", err)
		}

		var found models.Permission
		db.Where("name = ?", "user.view").First(&found)
		if found.DisplayName != "View Users Updated" {
			t.Errorf("Expected display name 'View Users Updated', got '%s'", found.DisplayName)
		}
	})

	t.Run("Register tenant-specific permission", func(t *testing.T) {
		tenantID := "tenant-1"
		perm := &models.Permission{
			ID:          uuid.New().String(),
			Name:        "tenant.data.view",
			Model:       "tenant-data",
			Action:      "view",
			DisplayName: "View Tenant Data",
			TenantID:    &tenantID,
		}

		err := permReg.Register(perm)
		if err != nil {
			t.Fatalf("Failed to register tenant permission: %v", err)
		}

		var found models.Permission
		db.Where("name = ? AND tenant_id = ?", "tenant.data.view", tenantID).First(&found)
		if found.TenantID == nil || *found.TenantID != tenantID {
			t.Errorf("Expected tenant ID '%s', got '%v'", tenantID, found.TenantID)
		}
	})

	t.Run("Register model permissions", func(t *testing.T) {
		err := permReg.RegisterModelPermissions("post", "Post", nil)
		if err != nil {
			t.Fatalf("Failed to register model permissions: %v", err)
		}

		var count int64
		db.Model(&models.Permission{}).Where("model = ?", "post").Count(&count)
		if count != 4 {
			t.Errorf("Expected 4 permissions for 'post' model, got %d", count)
		}
	})

	t.Run("Invalid permission name format", func(t *testing.T) {
		perm := &models.Permission{
			Name: "invalidname", // Missing dot
		}
		err := permReg.Register(perm)
		if err == nil {
			t.Error("Expected error for invalid permission name format")
		}
	})
}

func TestPermissionChecker(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(db)

	// Auto migrate
	db.AutoMigrate(&models.Permission{}, &models.Role{})

	// Create the role_user table if it doesn't exist
	db.Exec(`CREATE TABLE IF NOT EXISTS role_user (
		user_id VARCHAR(100),
		role_id VARCHAR(100),
		PRIMARY KEY (user_id, role_id)
	)`)

	permReg := permission.NewRegistrar(db)
	roleReg := role.NewRegistrar(db)

	// Create permissions
	permView := &models.Permission{
		ID:     uuid.New().String(),
		Name:   "user.view",
		Model:  "user",
		Action: "view",
	}
	permCreate := &models.Permission{
		ID:     uuid.New().String(),
		Name:   "user.create",
		Model:  "user",
		Action: "create",
	}
	permReg.Register(permView)
	permReg.Register(permCreate)

	// Create role with permissions
	roleAdmin := &models.Role{
		ID:          uuid.New().String(),
		Name:        "admin",
		DisplayName: "Administrator",
		Permissions: []models.Permission{*permView, *permCreate},
	}
	roleReg.Register(roleAdmin)

	// Assign role to user
	db.Exec("INSERT INTO role_user (user_id, role_id) VALUES (?, ?)", "user-1", roleAdmin.ID)

	checker := permission.NewChecker(db)
	ctx := context.Background()

	t.Run("Has permission", func(t *testing.T) {
		has, err := checker.HasPermission(ctx, "user-1", "user.view", nil)
		if err != nil {
			t.Fatalf("Failed to check permission: %v", err)
		}
		if !has {
			t.Error("Expected user to have 'user.view' permission")
		}
	})

	t.Run("Has model permission", func(t *testing.T) {
		has, err := checker.HasModelPermission(ctx, "user-1", "user", "view", nil)
		if err != nil {
			t.Fatalf("Failed to check model permission: %v", err)
		}
		if !has {
			t.Error("Expected user to have 'user.view' permission")
		}
	})

	t.Run("Has any permission", func(t *testing.T) {
		has, err := checker.HasAnyPermission(ctx, "user-1", nil, "user.view", "post.delete")
		if err != nil {
			t.Fatalf("Failed to check any permission: %v", err)
		}
		if !has {
			t.Error("Expected user to have at least one permission")
		}
	})

	t.Run("Has all permissions", func(t *testing.T) {
		has, err := checker.HasAllPermissions(ctx, "user-1", nil, "user.view", "user.create")
		if err != nil {
			t.Fatalf("Failed to check all permissions: %v", err)
		}
		if !has {
			t.Error("Expected user to have all permissions")
		}
	})

	t.Run("Get user permissions", func(t *testing.T) {
		perms, err := checker.GetUserPermissions(ctx, "user-1", nil)
		if err != nil {
			t.Fatalf("Failed to get user permissions: %v", err)
		}
		if len(perms) < 2 {
			t.Errorf("Expected at least 2 permissions, got %d", len(perms))
		}
	})

	t.Run("Get user permission names", func(t *testing.T) {
		names, err := checker.GetUserPermissionNames(ctx, "user-1", nil)
		if err != nil {
			t.Fatalf("Failed to get user permission names: %v", err)
		}
		if len(names) < 2 {
			t.Errorf("Expected at least 2 permission names, got %d", len(names))
		}
	})
}
