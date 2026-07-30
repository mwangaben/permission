package tests

import (
	"context"
	"testing"

	"github.com/mwangaben/permission/models"
	"github.com/mwangaben/permission/tenant"
)

func TestTenant(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(db)

	db.AutoMigrate(&models.Tenant{})

	db.Exec(`CREATE TABLE IF NOT EXISTS tenant_user (
		user_id VARCHAR(100),
		tenant_id VARCHAR(100)
	)`)

	resolver := tenant.NewResolver(db)

	t.Run("Create tenant", func(t *testing.T) {
		tenantObj, err := resolver.CreateTenant(
			"Test Organization",
			"test-org",
			"test.example.com",
			"user-1",
		)
		if err != nil {
			t.Fatalf("Failed to create tenant: %v", err)
		}

		if tenantObj.Name != "Test Organization" {
			t.Errorf("Expected name 'Test Organization', got '%s'", tenantObj.Name)
		}
		if tenantObj.Slug != "test-org" {
			t.Errorf("Expected slug 'test-org', got '%s'", tenantObj.Slug)
		}
		if tenantObj.Domain != "test.example.com" {
			t.Errorf("Expected domain 'test.example.com', got '%s'", tenantObj.Domain)
		}
		if tenantObj.OwnerID != "user-1" {
			t.Errorf("Expected owner ID 'user-1', got '%s'", tenantObj.OwnerID)
		}
	})

	t.Run("Resolve tenant by slug", func(t *testing.T) {
		tenantObj, err := resolver.CreateTenant(
			"Resolve Test",
			"resolve-test",
			"resolve.example.com",
			"user-2",
		)
		if err != nil {
			t.Fatalf("Failed to create tenant: %v", err)
		}

		resolved, err := resolver.ResolveTenantBySlug("resolve-test")
		if err != nil {
			t.Fatalf("Failed to resolve tenant: %v", err)
		}

		if resolved.ID != tenantObj.ID {
			t.Errorf("Expected tenant ID %s, got %s", tenantObj.ID, resolved.ID)
		}
	})

	t.Run("Resolve tenant by domain", func(t *testing.T) {
		tenantObj, err := resolver.CreateTenant(
			"Domain Test",
			"domain-test",
			"domain.example.com",
			"user-3",
		)
		if err != nil {
			t.Fatalf("Failed to create tenant: %v", err)
		}

		resolved, err := resolver.ResolveTenantByDomain("domain.example.com")
		if err != nil {
			t.Fatalf("Failed to resolve tenant by domain: %v", err)
		}

		if resolved.ID != tenantObj.ID {
			t.Errorf("Expected tenant ID %s, got %s", tenantObj.ID, resolved.ID)
		}
	})

	t.Run("Resolve tenant from context", func(t *testing.T) {
		tenantObj, err := resolver.CreateTenant(
			"Context Test",
			"context-test",
			"context.example.com",
			"user-4",
		)
		if err != nil {
			t.Fatalf("Failed to create tenant: %v", err)
		}

		ctx := context.WithValue(context.Background(), "tenant_id", tenantObj.ID)
		resolved, err := resolver.ResolveTenant(ctx)
		if err != nil {
			t.Fatalf("Failed to resolve tenant from context: %v", err)
		}

		if resolved.ID != tenantObj.ID {
			t.Errorf("Expected tenant ID %s, got %s", tenantObj.ID, resolved.ID)
		}
	})

	t.Run("Add user to tenant", func(t *testing.T) {
		tenantObj, err := resolver.CreateTenant(
			"Add User Test",
			"add-user-test",
			"adduser.example.com",
			"user-5",
		)
		if err != nil {
			t.Fatalf("Failed to create tenant: %v", err)
		}

		err = resolver.AddUserToTenant("user-6", tenantObj.ID)
		if err != nil {
			t.Fatalf("Failed to add user to tenant: %v", err)
		}

		var count int64
		db.Table("tenant_user").Where("user_id = ? AND tenant_id = ?", "user-6", tenantObj.ID).Count(&count)
		if count == 0 {
			t.Error("User was not added to tenant")
		}
	})

	t.Run("Remove user from tenant", func(t *testing.T) {
		tenantObj, err := resolver.CreateTenant(
			"Remove User Test",
			"remove-user-test",
			"removeuser.example.com",
			"user-7",
		)
		if err != nil {
			t.Fatalf("Failed to create tenant: %v", err)
		}

		resolver.AddUserToTenant("user-8", tenantObj.ID)

		err = resolver.RemoveUserFromTenant("user-8", tenantObj.ID)
		if err != nil {
			t.Fatalf("Failed to remove user from tenant: %v", err)
		}

		var count int64
		db.Table("tenant_user").Where("user_id = ? AND tenant_id = ?", "user-8", tenantObj.ID).Count(&count)
		if count > 0 {
			t.Error("User was not removed from tenant")
		}
	})

	t.Run("Get tenant users", func(t *testing.T) {
		tenantObj, err := resolver.CreateTenant(
			"Get Users Test",
			"get-users-test",
			"getusers.example.com",
			"user-9",
		)
		if err != nil {
			t.Fatalf("Failed to create tenant: %v", err)
		}

		users := []string{"user-10", "user-11", "user-12"}
		for _, userID := range users {
			resolver.AddUserToTenant(userID, tenantObj.ID)
		}

		userIDs, err := resolver.GetTenantUsers(tenantObj.ID)
		if err != nil {
			t.Fatalf("Failed to get tenant users: %v", err)
		}

		if len(userIDs) < 3 {
			t.Errorf("Expected at least 3 users, got %d", len(userIDs))
		}
	})

	t.Run("Get user tenants", func(t *testing.T) {
		tenants := []struct {
			name string
			slug string
		}{
			{"User Tenant 1", "user-tenant-1"},
			{"User Tenant 2", "user-tenant-2"},
		}

		for _, tData := range tenants {
			tenantObj, err := resolver.CreateTenant(
				tData.name,
				tData.slug,
				tData.slug+".example.com",
				"user-13",
			)
			if err != nil {
				t.Fatalf("Failed to create tenant: %v", err)
			}
			resolver.AddUserToTenant("user-13", tenantObj.ID)
		}

		userTenants, err := resolver.GetUserTenants("user-13")
		if err != nil {
			t.Fatalf("Failed to get user tenants: %v", err)
		}

		if len(userTenants) < 2 {
			t.Errorf("Expected at least 2 tenants, got %d", len(userTenants))
		}
	})
}
