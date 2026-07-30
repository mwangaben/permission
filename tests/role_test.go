package tests

import (
	"testing"

	"github.com/google/uuid"
	"github.com/mwangaben/permission/models"
	"github.com/mwangaben/permission/permission"
	"github.com/mwangaben/permission/role"
)

func TestRoleRegistration(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(db)

	db.AutoMigrate(&models.Permission{}, &models.Role{})

	permReg := permission.NewRegistrar(db)
	roleReg := role.NewRegistrar(db)

	// Create permissions
	perm1 := &models.Permission{
		ID:     uuid.New().String(),
		Name:   "user.view",
		Model:  "user",
		Action: "view",
	}
	perm2 := &models.Permission{
		ID:     uuid.New().String(),
		Name:   "user.edit",
		Model:  "user",
		Action: "edit",
	}
	permReg.Register(perm1)
	permReg.Register(perm2)

	t.Run("Register role", func(t *testing.T) {
		roleObj := &models.Role{
			ID:          uuid.New().String(),
			Name:        "editor",
			DisplayName: "Editor",
			Description: "Can view and edit users",
			Permissions: []models.Permission{*perm1, *perm2},
		}

		err := roleReg.Register(roleObj)
		if err != nil {
			t.Fatalf("Failed to register role: %v", err)
		}

		var found models.Role
		db.Where("name = ?", "editor").First(&found)
		if found.Name != "editor" {
			t.Errorf("Expected name 'editor', got '%s'", found.Name)
		}
	})

	t.Run("Register duplicate role (should update)", func(t *testing.T) {
		roleObj := &models.Role{
			Name:        "editor",
			DisplayName: "Editor Updated",
			Description: "Updated description",
			IsDefault:   true,
		}

		err := roleReg.Register(roleObj)
		if err != nil {
			t.Fatalf("Failed to register duplicate role: %v", err)
		}

		var found models.Role
		db.Where("name = ?", "editor").First(&found)
		if found.DisplayName != "Editor Updated" {
			t.Errorf("Expected display name 'Editor Updated', got '%s'", found.DisplayName)
		}
		if !found.IsDefault {
			t.Error("Expected role to be default")
		}
	})

	t.Run("Register tenant-specific role", func(t *testing.T) {
		tenantID := "tenant-1"
		roleObj := &models.Role{
			ID:          uuid.New().String(),
			Name:        "tenant-admin",
			DisplayName: "Tenant Admin",
			TenantID:    &tenantID,
			Permissions: []models.Permission{*perm1},
		}

		err := roleReg.Register(roleObj)
		if err != nil {
			t.Fatalf("Failed to register tenant role: %v", err)
		}

		var found models.Role
		db.Where("name = ? AND tenant_id = ?", "tenant-admin", tenantID).First(&found)
		if found.TenantID == nil || *found.TenantID != tenantID {
			t.Errorf("Expected tenant ID '%s', got '%v'", tenantID, found.TenantID)
		}
	})
}

func TestRoleAssignment(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(db)

	db.AutoMigrate(&models.Permission{}, &models.Role{})

	db.Exec(`CREATE TABLE IF NOT EXISTS role_user (
		user_id VARCHAR(100),
		role_id VARCHAR(100)
	)`)

	permReg := permission.NewRegistrar(db)
	roleReg := role.NewRegistrar(db)
	assigner := role.NewAssigner(db)

	perm := &models.Permission{
		ID:     uuid.New().String(),
		Name:   "user.view",
		Model:  "user",
		Action: "view",
	}
	permReg.Register(perm)

	roleObj := &models.Role{
		ID:          uuid.New().String(),
		Name:        "viewer",
		DisplayName: "Viewer",
		Permissions: []models.Permission{*perm},
	}
	roleReg.Register(roleObj)

	t.Run("Assign role to user", func(t *testing.T) {
		err := assigner.Assign("user-1", roleObj.ID)
		if err != nil {
			t.Fatalf("Failed to assign role: %v", err)
		}

		var count int64
		db.Table("role_user").Where("user_id = ? AND role_id = ?", "user-1", roleObj.ID).Count(&count)
		if count == 0 {
			t.Error("Role was not assigned to user")
		}
	})

	t.Run("Assign role by name", func(t *testing.T) {
		err := assigner.AssignByName("user-2", "viewer", nil)
		if err != nil {
			t.Fatalf("Failed to assign role by name: %v", err)
		}

		var count int64
		db.Table("role_user").Where("user_id = ?", "user-2").Count(&count)
		if count == 0 {
			t.Error("Role was not assigned to user by name")
		}
	})

	t.Run("Remove role from user", func(t *testing.T) {
		err := assigner.Remove("user-1", roleObj.ID)
		if err != nil {
			t.Fatalf("Failed to remove role: %v", err)
		}

		var count int64
		db.Table("role_user").Where("user_id = ? AND role_id = ?", "user-1", roleObj.ID).Count(&count)
		if count > 0 {
			t.Error("Role was not removed from user")
		}
	})

	t.Run("Sync roles", func(t *testing.T) {
		role2 := &models.Role{
			ID:          uuid.New().String(),
			Name:        "editor",
			DisplayName: "Editor",
		}
		roleReg.Register(role2)

		assigner.Assign("user-3", roleObj.ID)
		assigner.Assign("user-3", role2.ID)

		err := assigner.SyncRoles("user-3", []string{role2.ID})
		if err != nil {
			t.Fatalf("Failed to sync roles: %v", err)
		}

		var count int64
		db.Table("role_user").Where("user_id = ?", "user-3").Count(&count)
		if count != 1 {
			t.Errorf("Expected 1 role after sync, got %d", count)
		}
	})
}
