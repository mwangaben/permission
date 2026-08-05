package permission

import (
	"fmt"

	"github.com/mwangaben/permission/config"
	"github.com/mwangaben/permission/models"
	"github.com/mwangaben/permission/tenant"
	"gorm.io/gorm"
)

// Registrar handles permission registration with tenant support
type Registrar struct {
	db     *gorm.DB
	config *config.Config
	tenant *tenant.Manager
}

// NewRegistrar creates a new permission registrar
func NewRegistrar(db *gorm.DB, config *config.Config, tenant *tenant.Manager) *Registrar {
	return &Registrar{
		db:     db,
		config: config,
		tenant: tenant,
	}
}

// Register creates a new permission if it doesn't exist
func (r *Registrar) Register(name string, guardName string) (*models.Permission, error) {
	if guardName == "" {
		guardName = r.config.DefaultGuard
	}

	var permission models.Permission
	var err error

	if r.tenant.IsEnabled() {
		tenantID := r.tenant.GetTenantID()
		err = r.db.Where("name = ? AND guard_name = ? AND tenant_id = ?", name, guardName, tenantID).First(&permission).Error
		if err == nil {
			return &permission, nil
		}
		permission = models.Permission{
			Name:      name,
			GuardName: guardName,
			TenantID:  &tenantID,
		}
	} else {
		err = r.db.Where("name = ? AND guard_name = ? AND tenant_id IS NULL", name, guardName).First(&permission).Error
		if err == nil {
			return &permission, nil
		}
		permission = models.Permission{
			Name:      name,
			GuardName: guardName,
			TenantID:  nil,
		}
	}

	if err := r.db.Create(&permission).Error; err != nil {
		return nil, fmt.Errorf("failed to create permission: %w", err)
	}

	return &permission, nil
}

// RegisterGlobal creates a global permission (available to all tenants)
func (r *Registrar) RegisterGlobal(name string, guardName string) (*models.Permission, error) {
	if guardName == "" {
		guardName = r.config.DefaultGuard
	}

	var permission models.Permission
	err := r.db.Where("name = ? AND guard_name = ? AND tenant_id IS NULL", name, guardName).First(&permission).Error
	if err == nil {
		return &permission, nil
	}

	permission = models.Permission{
		Name:      name,
		GuardName: guardName,
		TenantID:  nil,
	}

	if err := r.db.Create(&permission).Error; err != nil {
		return nil, fmt.Errorf("failed to create global permission: %w", err)
	}

	return &permission, nil
}

// RegisterMany registers multiple permissions
func (r *Registrar) RegisterMany(permissions []struct{ Name, GuardName string }) ([]models.Permission, error) {
	var created []models.Permission
	for _, p := range permissions {
		perm, err := r.Register(p.Name, p.GuardName)
		if err != nil {
			return nil, err
		}
		created = append(created, *perm)
	}
	return created, nil
}

// FindByName finds a permission by name
func (r *Registrar) FindByName(name string, guardName string) (*models.Permission, error) {
	if guardName == "" {
		guardName = r.config.DefaultGuard
	}

	var permission models.Permission
	var err error

	if r.tenant.IsEnabled() {
		tenantID := r.tenant.GetTenantID()
		err = r.db.Where("name = ? AND guard_name = ? AND (tenant_id = ? OR tenant_id IS NULL)", name, guardName, tenantID).First(&permission).Error
	} else {
		err = r.db.Where("name = ? AND guard_name = ?", name, guardName).First(&permission).Error
	}

	if err != nil {
		return nil, err
	}
	return &permission, nil
}
