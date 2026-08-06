package permission

import (
	"github.com/mwangaben/permission/config"
	"github.com/mwangaben/permission/models"
	"github.com/mwangaben/permission/role"
	"github.com/mwangaben/permission/tenant"
	"gorm.io/gorm"
)

// Manager is the main entry point for the permission package
type Manager struct {
	DB        *gorm.DB
	Config    *config.Config
	Tenant    *tenant.Manager
	Registrar *Registrar
	Checker   *Checker
	Guard     *Guard
	// Role management
	RoleRegistrar   *role.Registrar
	RoleAssigner    *role.Assigner
	RolePermManager *role.PermissionManager
}

// NewManager creates a new permission manager
func NewManager(db *gorm.DB, opts ...func(*config.Config)) *Manager {
	cfg := config.NewConfig(opts...)
	tenantManager := tenant.NewManager(cfg.EnableTenant)

	registrar := NewRegistrar(db, cfg, tenantManager)
	checker := NewChecker(db, cfg, tenantManager)
	guard := NewGuard(checker)

	roleRegistrar := role.NewRegistrar(db, cfg, tenantManager)
	roleAssigner := role.NewAssigner(db)
	rolePermManager := role.NewPermissionManager(db)

	return &Manager{
		DB:              db,
		Config:          cfg,
		Tenant:          tenantManager,
		Registrar:       registrar,
		Checker:         checker,
		Guard:           guard,
		RoleRegistrar:   roleRegistrar,
		RoleAssigner:    roleAssigner,
		RolePermManager: rolePermManager,
	}
}

// WithTenant sets the tenant context for all operations
func (m *Manager) WithTenant(tenantID string) *Manager {
	m.Tenant = m.Tenant.WithTenant(tenantID)
	m.Registrar = NewRegistrar(m.DB, m.Config, m.Tenant)
	m.Checker = NewChecker(m.DB, m.Config, m.Tenant)
	m.Guard = NewGuard(m.Checker)
	m.RoleRegistrar = role.NewRegistrar(m.DB, m.Config, m.Tenant)
	return m
}

// EnableTenant enables tenant mode
func (m *Manager) EnableTenant(tenantIDType string) *Manager {
	m.Config.EnableTenant = true
	m.Config.TenantIDType = tenantIDType
	m.Tenant = tenant.NewManager(true)
	m.Registrar = NewRegistrar(m.DB, m.Config, m.Tenant)
	m.Checker = NewChecker(m.DB, m.Config, m.Tenant)
	m.Guard = NewGuard(m.Checker)
	m.RoleRegistrar = role.NewRegistrar(m.DB, m.Config, m.Tenant)
	return m
}

// DisableTenant disables tenant mode
func (m *Manager) DisableTenant() *Manager {
	m.Config.EnableTenant = false
	m.Tenant = tenant.NewManager(false)
	m.Registrar = NewRegistrar(m.DB, m.Config, m.Tenant)
	m.Checker = NewChecker(m.DB, m.Config, m.Tenant)
	m.Guard = NewGuard(m.Checker)
	m.RoleRegistrar = role.NewRegistrar(m.DB, m.Config, m.Tenant)
	return m
}

// Migrate runs database migrations
func (m *Manager) Migrate() error {
	return m.DB.AutoMigrate(
		&models.Permission{},
		&models.Role{},
		&models.RoleHasPermission{},
		&models.ModelHasRole{},
		&models.ModelHasPermission{},
	)
}

// SeedDefaultPermissions seeds default permissions
func (m *Manager) SeedDefaultPermissions() error {
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

	_, err := m.Registrar.RegisterMany(permissions)
	return err
}

// SeedDefaultRoles seeds default roles
func (m *Manager) SeedDefaultRoles() error {
	// Register default roles
	roles := []struct {
		Name        string
		GuardName   string
		Permissions []string
	}{
		{
			Name:        "super-admin",
			GuardName:   "web",
			Permissions: []string{"user.view", "user.create", "user.update", "user.delete", "post.view", "post.create", "post.update", "post.delete", "report.view", "report.create", "report.update", "report.delete"},
		},
		{
			Name:        "admin",
			GuardName:   "web",
			Permissions: []string{"user.view", "user.create", "user.update", "user.delete", "post.view", "post.create", "post.update", "post.delete", "report.view", "report.create", "report.update", "report.delete"},
		},
		{
			Name:        "editor",
			GuardName:   "web",
			Permissions: []string{"post.view", "post.create", "post.update", "post.delete", "report.view", "report.create"},
		},
		{
			Name:        "viewer",
			GuardName:   "web",
			Permissions: []string{"user.view", "post.view", "report.view"},
		},
	}

	for _, r := range roles {
		roleObj, err := m.RoleRegistrar.Register(r.Name, r.GuardName)
		if err != nil {
			return err
		}

		for _, permName := range r.Permissions {
			perm, err := m.Registrar.FindByName(permName, r.GuardName)
			if err != nil {
				continue
			}
			if err := m.RolePermManager.AssignPermissionToRole(perm.ID, roleObj.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetUserPermissions returns all permissions for a user (model)
func (m *Manager) GetUserPermissions(modelType string, modelID uint, guardName string) ([]models.Permission, error) {
	if guardName == "" {
		guardName = m.Config.DefaultGuard
	}

	checker := NewChecker(m.DB, m.Config, m.Tenant)
	return checker.GetAllPermissionsForModel(modelType, modelID, guardName)
}

// HasUserPermission checks if a user has a specific permission
func (m *Manager) HasUserPermission(modelType string, modelID uint, permissionName, guardName string) (bool, error) {
	if guardName == "" {
		guardName = m.Config.DefaultGuard
	}

	checker := NewChecker(m.DB, m.Config, m.Tenant)
	return checker.HasPermission(modelType, modelID, permissionName, guardName)
}

// AssignRoleToUser assigns a role to a user
func (m *Manager) AssignRoleToUser(roleName, modelType string, modelID uint, guardName string) error {
	if guardName == "" {
		guardName = m.Config.DefaultGuard
	}

	return m.RoleAssigner.AssignRoleToModelByName(roleName, modelType, modelID, guardName)
}

// RemoveRoleFromUser removes a role from a user
func (m *Manager) RemoveRoleFromUser(roleName, modelType string, modelID uint, guardName string) error {
	if guardName == "" {
		guardName = m.Config.DefaultGuard
	}

	roleObj, err := m.RoleRegistrar.FindByName(roleName, guardName)
	if err != nil {
		return err
	}

	return m.RoleAssigner.RemoveRoleFromModel(roleObj.ID, modelType, modelID)
}
