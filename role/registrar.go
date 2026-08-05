package role

import (
	"fmt"

	"github.com/mwangaben/permission/config"
	"github.com/mwangaben/permission/models"
	"github.com/mwangaben/permission/tenant"
	"gorm.io/gorm"
)

// Registrar handles role registration with tenant support
type Registrar struct {
	db     *gorm.DB
	config *config.Config
	tenant *tenant.Manager
}

// NewRegistrar creates a new role registrar
func NewRegistrar(db *gorm.DB, config *config.Config, tenant *tenant.Manager) *Registrar {
	return &Registrar{
		db:     db,
		config: config,
		tenant: tenant,
	}
}

// Register creates a new role if it doesn't exist
func (r *Registrar) Register(name string, guardName string) (*models.Role, error) {
	if guardName == "" {
		guardName = r.config.DefaultGuard
	}

	var role models.Role
	var err error

	if r.tenant.IsEnabled() {
		tenantID := r.tenant.GetTenantID()
		err = r.db.Where("name = ? AND guard_name = ? AND tenant_id = ?", name, guardName, tenantID).First(&role).Error
		if err == nil {
			// Load permissions for the role
			r.db.Preload("Permissions").First(&role, role.ID)
			return &role, nil
		}
		role = models.Role{
			Name:      name,
			GuardName: guardName,
			TenantID:  &tenantID,
		}
	} else {
		err = r.db.Where("name = ? AND guard_name = ?", name, guardName).First(&role).Error
		if err == nil {
			// Load permissions for the role
			r.db.Preload("Permissions").First(&role, role.ID)
			return &role, nil
		}
		role = models.Role{
			Name:      name,
			GuardName: guardName,
			TenantID:  nil,
		}
	}

	if err := r.db.Create(&role).Error; err != nil {
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	return &role, nil
}

// RegisterGlobal creates a global role (available to all tenants)
func (r *Registrar) RegisterGlobal(name string, guardName string) (*models.Role, error) {
	if guardName == "" {
		guardName = r.config.DefaultGuard
	}

	var role models.Role
	err := r.db.Where("name = ? AND guard_name = ? AND tenant_id IS NULL", name, guardName).First(&role).Error
	if err == nil {
		return &role, nil
	}

	role = models.Role{
		Name:      name,
		GuardName: guardName,
		TenantID:  nil,
	}

	if err := r.db.Create(&role).Error; err != nil {
		return nil, fmt.Errorf("failed to create global role: %w", err)
	}

	return &role, nil
}

// FindByName finds a role by name
func (r *Registrar) FindByName(name string, guardName string) (*models.Role, error) {
	if guardName == "" {
		guardName = r.config.DefaultGuard
	}

	var role models.Role
	var err error

	if r.tenant.IsEnabled() {
		tenantID := r.tenant.GetTenantID()
		err = r.db.Where("name = ? AND guard_name = ? AND (tenant_id = ? OR tenant_id IS NULL)", name, guardName, tenantID).First(&role).Error
	} else {
		err = r.db.Where("name = ? AND guard_name = ?", name, guardName).First(&role).Error
	}

	if err != nil {
		return nil, err
	}

	// Load permissions for the role
	if err := r.db.Preload("Permissions").First(&role, role.ID).Error; err != nil {
		return nil, err
	}

	return &role, nil
}
