package permission

import (
	"context"
	"errors"
	"fmt"

	"github.com/mwangaben/permission/config"
	"github.com/mwangaben/permission/role"
	"github.com/mwangaben/permission/storage"
	"github.com/mwangaben/permission/storage/entstore"
	"github.com/mwangaben/permission/storage/gormstore"
	"github.com/mwangaben/permission/tenant"
)

// Manager is the main entry point for the permission package.
//
// It works with either a *gorm.DB or an *ent.Client. The backend is
// auto-detected from the type of the argument passed to NewManager.
type Manager struct {
	Repo      storage.Repository
	Config    *config.Config
	Tenant    *tenant.Manager
	Registrar *Registrar
	Checker   *Checker
	Guard     *Guard

	RoleRegistrar   *role.Registrar
	RoleAssigner    *role.Assigner
	RolePermManager *role.PermissionManager
}

// NewManager creates a new manager. db may be *gorm.DB, *ent.Client, or any
// value exposing DB() *gorm.DB or Client() *ent.Client.
func NewManager(db interface{}, opts ...func(*config.Config)) (*Manager, error) {
	cfg := config.NewConfig(opts...)
	tenantManager := tenant.NewManager(cfg.EnableTenant)

	repo, err := buildRepository(db, "")
	if err != nil {
		return nil, fmt.Errorf("failed to build storage: %w", err)
	}

	m := &Manager{
		Repo:   repo,
		Config: cfg,
		Tenant: tenantManager,
	}
	m.wire()
	return m, nil
}

// wire constructs the sub-components from the current Repo/Config/Tenant.
// Called by NewManager and any method that changes tenant state.
func (m *Manager) wire() {
	m.Registrar = NewRegistrar(m.Repo, m.Config, m.Tenant)
	m.Checker = NewChecker(m.Repo, m.Config, m.Tenant)
	m.Guard = NewGuard(m.Checker)
	m.RoleRegistrar = role.NewRegistrar(m.Repo, m.Config, m.Tenant)
	m.RoleAssigner = role.NewAssigner(m.Repo)
	m.RolePermManager = role.NewPermissionManager(m.Repo)
}

func buildRepository(db interface{}, forceDriver string) (storage.Repository, error) {
	driver := forceDriver
	if driver == "" {
		var err error
		driver, err = storage.DetectDriver(db)
		if err != nil {
			return nil, err
		}
	}
	switch driver {
	case storage.DriverGorm:
		return gormstore.New(db)
	case storage.DriverEnt:
		return entstore.New(db)
	default:
		return nil, fmt.Errorf("unsupported storage driver: %s", driver)
	}
}

func (m *Manager) WithTenant(tenantID string) *Manager {
	m.Tenant = m.Tenant.WithTenant(tenantID)
	m.wire()
	return m
}

func (m *Manager) EnableTenant(tenantIDType string) *Manager {
	m.Config.EnableTenant = true
	m.Config.TenantIDType = tenantIDType
	m.Tenant = tenant.NewManager(true)
	m.wire()
	return m
}

func (m *Manager) DisableTenant() *Manager {
	m.Config.EnableTenant = false
	m.Tenant = tenant.NewManager(false)
	m.wire()
	return m
}

// Migrate runs schema migrations for the active backend.
func (m *Manager) Migrate(ctx context.Context) error {
	return m.Repo.AutoMigrate(ctx)
}

// SeedDefaultPermissions seeds the default permission set.
func (m *Manager) SeedDefaultPermissions(ctx context.Context) error {
	permissions := []struct{ Name, GuardName string }{
		{"user.view", "web"},
		{"user.create", "web"},
		{"user.update", "web"},
		{"user.delete", "web"},
		{"post.view", "web"},
		{"post.create", "web"},
		{"post.update", "web"},
		{"post.delete", "web"},
		{"report.view", "web"},
		{"report.create", "web"},
		{"report.update", "web"},
		{"report.delete", "web"},
	}
	_, err := m.Registrar.RegisterMany(ctx, permissions)
	return err
}

// SeedDefaultRoles seeds the default role set with their permissions.
func (m *Manager) SeedDefaultRoles(ctx context.Context) error {
	roleDefs := []struct {
		Name        string
		GuardName   string
		Permissions []string
	}{
		{Name: "super-admin", GuardName: "web", Permissions: []string{ /* ... */ }},
		{Name: "admin", GuardName: "web", Permissions: []string{ /* ... */ }},
		{Name: "editor", GuardName: "web", Permissions: []string{ /* ... */ }},
		{Name: "viewer", GuardName: "web", Permissions: []string{ /* ... */ }},
	}

	for _, r := range roleDefs {
		roleObj, err := m.RoleRegistrar.Register(ctx, r.Name, r.GuardName)
		if err != nil {
			return err
		}
		for _, permName := range r.Permissions {
			perm, err := m.Registrar.FindByName(ctx, permName, r.GuardName)
			if err != nil {
				if errors.Is(err, storage.ErrPermissionNotFound) {
					continue
				}
				return err
			}
			if err := m.RolePermManager.AssignPermissionToRole(ctx, perm.ID, roleObj.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *Manager) GetUserPermissions(ctx context.Context, modelType string, modelID uint, guardName string) ([]*storage.Permission, error) {
	if guardName == "" {
		guardName = m.Config.DefaultGuard
	}
	return m.Checker.GetAllPermissionsForModel(ctx, modelType, modelID, guardName)
}

func (m *Manager) HasUserPermission(ctx context.Context, modelType string, modelID uint, permissionName, guardName string) (bool, error) {
	if guardName == "" {
		guardName = m.Config.DefaultGuard
	}
	return m.Checker.HasPermission(ctx, modelType, modelID, permissionName, guardName)
}

func (m *Manager) AssignRoleToUser(ctx context.Context, roleName, modelType string, modelID uint, guardName string) error {
	if guardName == "" {
		guardName = m.Config.DefaultGuard
	}
	return m.RoleAssigner.AssignRoleToModelByName(ctx, roleName, modelType, modelID, guardName, m.currentTenantID())
}

func (m *Manager) RemoveRoleFromUser(ctx context.Context, roleName, modelType string, modelID uint, guardName string) error {
	if guardName == "" {
		guardName = m.Config.DefaultGuard
	}
	roleObj, err := m.RoleRegistrar.FindByName(ctx, roleName, guardName)
	if err != nil {
		return err
	}
	return m.RoleAssigner.RemoveRoleFromModel(ctx, roleObj.ID, modelType, modelID)
}

// currentTenantID returns the tenant ID to use for write operations, or nil
// if tenant mode is disabled.
func (m *Manager) currentTenantID() *string {
	if m.Tenant == nil || !m.Tenant.IsEnabled() {
		return nil
	}
	tid := m.Tenant.GetTenantID()
	if tid == "" {
		return nil
	}
	return &tid
}
