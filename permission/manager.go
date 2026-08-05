package permission

import (
	"github.com/mwangaben/permission/config"
	"github.com/mwangaben/permission/models"
	"github.com/mwangaben/permission/tenant"
	"gorm.io/gorm"
)

// PermManager is the main entry point for the permission package
type PermManager struct {
	DB        *gorm.DB
	Config    *config.Config
	Tenant    *tenant.Manager
	Registrar *Registrar
	Checker   *Checker
	Guard     *Guard
}

// NewPermManager creates a new permission manager
func NewPermManager(db *gorm.DB, opts ...func(*config.Config)) *PermManager {
	cfg := config.NewConfig(opts...)
	tenantManager := tenant.NewManager(cfg.EnableTenant)

	registrar := NewRegistrar(db, cfg, tenantManager)
	checker := NewChecker(db, cfg, tenantManager)
	guard := NewGuard(checker)

	return &PermManager{
		DB:        db,
		Config:    cfg,
		Tenant:    tenantManager,
		Registrar: registrar,
		Checker:   checker,
		Guard:     guard,
	}
}

// WithTenant sets the tenant context for all operations
func (m *PermManager) WithTenant(tenantID string) *PermManager {
	m.Tenant = m.Tenant.WithTenant(tenantID)
	m.Registrar = NewRegistrar(m.DB, m.Config, m.Tenant)
	m.Checker = NewChecker(m.DB, m.Config, m.Tenant)
	m.Guard = NewGuard(m.Checker)
	return m
}

// EnableTenant enables tenant mode
func (m *PermManager) EnableTenant(tenantIDType string) *PermManager {
	m.Config.EnableTenant = true
	m.Config.TenantIDType = tenantIDType
	m.Tenant = tenant.NewManager(true)
	m.Registrar = NewRegistrar(m.DB, m.Config, m.Tenant)
	m.Checker = NewChecker(m.DB, m.Config, m.Tenant)
	m.Guard = NewGuard(m.Checker)
	return m
}

// DisableTenant disables tenant mode
func (m *PermManager) DisableTenant() *PermManager {
	m.Config.EnableTenant = false
	m.Tenant = tenant.NewManager(false)
	m.Registrar = NewRegistrar(m.DB, m.Config, m.Tenant)
	m.Checker = NewChecker(m.DB, m.Config, m.Tenant)
	m.Guard = NewGuard(m.Checker)
	return m
}

// Migrate runs database migrations
func (m *PermManager) Migrate() error {
	return m.DB.AutoMigrate(
		&models.Permission{},
		&models.Role{},
		&models.RoleHasPermission{},
		&models.ModelHasRole{},
		&models.ModelHasPermission{},
	)
}

// SeedDefaultPermissions seeds default permissions
func (m *PermManager) SeedDefaultPermissions() error {
	permissions := []struct{ Name, GuardName string }{
		{"user.view", "web"},
		{"user.create", "web"},
		{"user.update", "web"},
		{"user.delete", "web"},
	}

	_, err := m.Registrar.RegisterMany(permissions)
	return err
}

// SeedDefaultRoles seeds default roles
func (m *PermManager) SeedDefaultRoles() error {
	// This would require a role registrar with permission assignment
	// Implementation depends on your specific needs
	return nil
}
