package tests

import (
	"context"
	"testing"

	"github.com/mwangaben/permission/gate"
)

func TestGate(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(db)

	g := gate.NewGate(&gate.Config{DB: db})

	// Register ability
	g.RegisterAbility("view-dashboard", func(ctx context.Context, user interface{}, args ...interface{}) bool {
		return user != nil
	})

	// Register ability with response
	g.RegisterAbilityWithResponse("view-admin", func(ctx context.Context, user interface{}, args ...interface{}) *gate.Response {
		if user == nil {
			return gate.NewDenied("User not authenticated")
		}
		if u, ok := user.(map[string]string); ok {
			if u["role"] != "admin" {
				return gate.NewDenied("Admin role required")
			}
		}
		return gate.NewAllowed()
	})

	ctx := context.Background()
	user := map[string]string{"id": "1", "name": "Test User"}

	t.Run("Allows with valid user", func(t *testing.T) {
		if !g.Allows(ctx, user, "view-dashboard") {
			t.Error("Expected user to be allowed to view dashboard")
		}
	})

	t.Run("Denies with nil user", func(t *testing.T) {
		if g.Allows(ctx, nil, "view-dashboard") {
			t.Error("Expected nil user to be denied")
		}
	})

	t.Run("Allows admin user", func(t *testing.T) {
		adminUser := map[string]string{"id": "1", "name": "Admin", "role": "admin"}
		if !g.Allows(ctx, adminUser, "view-admin") {
			t.Error("Expected admin user to be allowed")
		}
	})

	t.Run("Denies non-admin user", func(t *testing.T) {
		nonAdminUser := map[string]string{"id": "2", "name": "User", "role": "user"}
		if g.Allows(ctx, nonAdminUser, "view-admin") {
			t.Error("Expected non-admin user to be denied")
		}
	})

	t.Run("Check returns full response", func(t *testing.T) {
		response := g.Check(ctx, user, "view-dashboard")
		if !response.IsAllowed() {
			t.Error("Expected response to be allowed")
		}
	})

	t.Run("Check returns denied response", func(t *testing.T) {
		response := g.Check(ctx, nil, "view-dashboard")
		if response.IsDenied() {
			t.Logf("Denied response: %s", response.Reason)
		} else {
			t.Error("Expected response to be denied")
		}
	})
}

func TestGateWithPolicies(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(db)

	g := gate.NewGate(&gate.Config{DB: db})

	view := func(ctx context.Context, user interface{}, resource interface{}) bool {
		return user != nil
	}
	update := func(ctx context.Context, user interface{}, resource interface{}) bool {
		if user == nil {
			return false
		}
		if u, ok := user.(map[string]string); ok {
			return u["role"] == "admin"
		}
		return false
	}

	g.RegisterAbility("TestPolicy@view", func(ctx context.Context, user interface{}, args ...interface{}) bool {
		if len(args) > 0 {
			return view(ctx, user, args[0])
		}
		return view(ctx, user, nil)
	})

	g.RegisterAbility("TestPolicy@update", func(ctx context.Context, user interface{}, args ...interface{}) bool {
		if len(args) > 0 {
			return update(ctx, user, args[0])
		}
		return update(ctx, user, nil)
	})

	ctx := context.Background()
	user := map[string]string{"id": "1", "name": "Test User", "role": "user"}
	adminUser := map[string]string{"id": "2", "name": "Admin", "role": "admin"}

	t.Run("View policy with user", func(t *testing.T) {
		if !g.Allows(ctx, user, "TestPolicy@view", "resource-1") {
			t.Error("Expected user to be allowed to view")
		}
	})

	t.Run("View policy without user", func(t *testing.T) {
		if g.Allows(ctx, nil, "TestPolicy@view", "resource-1") {
			t.Error("Expected nil user to be denied")
		}
	})

	t.Run("Update policy with admin user", func(t *testing.T) {
		if !g.Allows(ctx, adminUser, "TestPolicy@update", "resource-1") {
			t.Error("Expected admin user to be allowed to update")
		}
	})

	t.Run("Update policy with non-admin user", func(t *testing.T) {
		if g.Allows(ctx, user, "TestPolicy@update", "resource-1") {
			t.Error("Expected non-admin user to be denied update")
		}
	})
}

func TestGateWithHooks(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(db)

	g := gate.NewGate(&gate.Config{DB: db})

	// Register before hook - allows super admin
	g.Before(func(ctx context.Context, user interface{}, ability string, arguments ...interface{}) *gate.Response {
		if user == nil {
			return gate.NewDenied("Authentication required")
		}
		if u, ok := user.(map[string]string); ok && u["role"] == "super-admin" {
			return gate.NewAllowed()
		}
		return nil
	})

	// Register after hook - denies with message if not allowed
	// This runs after the ability check
	g.After(func(ctx context.Context, user interface{}, ability string, result bool, arguments ...interface{}) *gate.Response {
		// If the ability check returned false, deny with message
		if !result {
			return &gate.Response{
				Allowed: false,
				Reason:  "Access denied",
				Message: "You do not have permission to perform this action",
			}
		}
		// If the ability check returned true, we still want to check if the user is regular
		// For regular users, we want to deny access with a message
		if u, ok := user.(map[string]string); ok && u["role"] == "user" {
			return &gate.Response{
				Allowed: false,
				Reason:  "Access denied",
				Message: "You do not have permission to perform this action",
			}
		}
		return nil
	})

	// Register a simple ability that always returns true for regular users
	// For the test, we want it to return true so the after hook can deny it
	g.RegisterAbility("view-settings", func(ctx context.Context, user interface{}, args ...interface{}) bool {
		if user == nil {
			return false
		}
		// Allow super admin, deny regular user
		if u, ok := user.(map[string]string); ok {
			if u["role"] == "super-admin" {
				return true
			}
			if u["role"] == "user" {
				return false // Let the after hook handle the denial
			}
		}
		return true
	})

	ctx := context.Background()
	superAdmin := map[string]string{"id": "1", "name": "Super Admin", "role": "super-admin"}
	regularUser := map[string]string{"id": "2", "name": "Regular User", "role": "user"}

	t.Run("Super admin bypasses all checks", func(t *testing.T) {
		response := g.Check(ctx, superAdmin, "view-settings")
		if !response.IsAllowed() {
			t.Error("Expected super admin to bypass all checks")
		}
	})

	t.Run("Regular user gets denied with message", func(t *testing.T) {
		response := g.Check(ctx, regularUser, "view-settings")
		if response.Allowed {
			t.Error("Expected regular user to be denied")
		}
		if response.Message != "You do not have permission to perform this action" {
			t.Errorf("Expected message 'You do not have permission to perform this action', got '%s'", response.Message)
		}
	})

	t.Run("Nil user gets authentication required", func(t *testing.T) {
		response := g.Check(ctx, nil, "view-settings")
		if response.Allowed {
			t.Error("Expected nil user to be denied")
		}
		if response.Reason != "Authentication required" {
			t.Errorf("Expected reason 'Authentication required', got '%s'", response.Reason)
		}
	})
}
