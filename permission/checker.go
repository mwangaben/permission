package permission

import (
	"github.com/mwangaben/permission/config"
	"github.com/mwangaben/permission/models"
	"github.com/mwangaben/permission/tenant"
	"gorm.io/gorm"
)

// Checker handles permission checking with tenant support
type Checker struct {
	db     *gorm.DB
	config *config.Config
	tenant *tenant.Manager
}

// NewChecker creates a new permission checker
func NewChecker(db *gorm.DB, config *config.Config, tenant *tenant.Manager) *Checker {
	return &Checker{
		db:     db,
		config: config,
		tenant: tenant,
	}
}

// HasPermission checks if a model has a specific permission
func (c *Checker) HasPermission(modelType string, modelID uint, permissionName string, guardName string) (bool, error) {
	if guardName == "" {
		guardName = c.config.DefaultGuard
	}

	var count int64

	// Build base query
	query := c.db.Table("permissions").
		Joins("JOIN model_has_permissions ON model_has_permissions.permission_id = permissions.id").
		Where("permissions.name = ? AND permissions.guard_name = ?", permissionName, guardName).
		Where("model_has_permissions.model_type = ? AND model_has_permissions.model_id = ?", modelType, modelID)

	// Add tenant scoping if enabled
	if c.tenant.IsEnabled() {
		tenantID := c.tenant.GetTenantID()
		query = query.Where("(permissions.tenant_id = ? OR permissions.tenant_id IS NULL)", tenantID)
		query = query.Where("(model_has_permissions.tenant_id = ? OR model_has_permissions.tenant_id IS NULL)", tenantID)
	}

	err := query.Count(&count).Error
	if err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}

	// Check permissions via roles
	query = c.db.Table("permissions").
		Joins("JOIN role_has_permissions ON role_has_permissions.permission_id = permissions.id").
		Joins("JOIN roles ON roles.id = role_has_permissions.role_id").
		Joins("JOIN model_has_roles ON model_has_roles.role_id = roles.id").
		Where("permissions.name = ? AND permissions.guard_name = ?", permissionName, guardName).
		Where("model_has_roles.model_type = ? AND model_has_roles.model_id = ?", modelType, modelID)

	if c.tenant.IsEnabled() {
		tenantID := c.tenant.GetTenantID()
		query = query.Where("(permissions.tenant_id = ? OR permissions.tenant_id IS NULL)", tenantID)
		query = query.Where("(roles.tenant_id = ? OR roles.tenant_id IS NULL)", tenantID)
		query = query.Where("(role_has_permissions.tenant_id = ? OR role_has_permissions.tenant_id IS NULL)", tenantID)
		query = query.Where("(model_has_roles.tenant_id = ? OR model_has_roles.tenant_id IS NULL)", tenantID)
	}

	err = query.Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// HasAnyPermission checks if a model has any of the given permissions
func (c *Checker) HasAnyPermission(modelType string, modelID uint, guardName string, permissionNames ...string) (bool, error) {
	if guardName == "" {
		guardName = c.config.DefaultGuard
	}

	if len(permissionNames) == 0 {
		return false, nil
	}

	var count int64

	query := c.db.Table("permissions").
		Joins("JOIN model_has_permissions ON model_has_permissions.permission_id = permissions.id").
		Where("permissions.name IN (?) AND permissions.guard_name = ?", permissionNames, guardName).
		Where("model_has_permissions.model_type = ? AND model_has_permissions.model_id = ?", modelType, modelID)

	if c.tenant.IsEnabled() {
		tenantID := c.tenant.GetTenantID()
		query = query.Where("(permissions.tenant_id = ? OR permissions.tenant_id IS NULL)", tenantID)
		query = query.Where("(model_has_permissions.tenant_id = ? OR model_has_permissions.tenant_id IS NULL)", tenantID)
	}

	err := query.Count(&count).Error
	if err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}

	query = c.db.Table("permissions").
		Joins("JOIN role_has_permissions ON role_has_permissions.permission_id = permissions.id").
		Joins("JOIN roles ON roles.id = role_has_permissions.role_id").
		Joins("JOIN model_has_roles ON model_has_roles.role_id = roles.id").
		Where("permissions.name IN (?) AND permissions.guard_name = ?", permissionNames, guardName).
		Where("model_has_roles.model_type = ? AND model_has_roles.model_id = ?", modelType, modelID)

	if c.tenant.IsEnabled() {
		tenantID := c.tenant.GetTenantID()
		query = query.Where("(permissions.tenant_id = ? OR permissions.tenant_id IS NULL)", tenantID)
		query = query.Where("(roles.tenant_id = ? OR roles.tenant_id IS NULL)", tenantID)
		query = query.Where("(role_has_permissions.tenant_id = ? OR role_has_permissions.tenant_id IS NULL)", tenantID)
		query = query.Where("(model_has_roles.tenant_id = ? OR model_has_roles.tenant_id IS NULL)", tenantID)
	}

	err = query.Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetAllPermissionsForModel returns all permissions for a model
func (c *Checker) GetAllPermissionsForModel(modelType string, modelID uint, guardName string) ([]models.Permission, error) {
	if guardName == "" {
		guardName = c.config.DefaultGuard
	}

	// Get direct permissions
	var directPerms []models.Permission
	query := c.db.Table("permissions").
		Joins("JOIN model_has_permissions ON model_has_permissions.permission_id = permissions.id").
		Where("permissions.guard_name = ?", guardName).
		Where("model_has_permissions.model_type = ? AND model_has_permissions.model_id = ?", modelType, modelID)

	if c.tenant.IsEnabled() {
		tenantID := c.tenant.GetTenantID()
		query = query.Where("(permissions.tenant_id = ? OR permissions.tenant_id IS NULL)", tenantID)
		query = query.Where("(model_has_permissions.tenant_id = ? OR model_has_permissions.tenant_id IS NULL)", tenantID)
	}

	if err := query.Find(&directPerms).Error; err != nil {
		return nil, err
	}

	// Get permissions via roles
	var rolePerms []models.Permission
	query = c.db.Table("permissions").
		Joins("JOIN role_has_permissions ON role_has_permissions.permission_id = permissions.id").
		Joins("JOIN roles ON roles.id = role_has_permissions.role_id").
		Joins("JOIN model_has_roles ON model_has_roles.role_id = roles.id").
		Where("permissions.guard_name = ?", guardName).
		Where("model_has_roles.model_type = ? AND model_has_roles.model_id = ?", modelType, modelID)

	if c.tenant.IsEnabled() {
		tenantID := c.tenant.GetTenantID()
		query = query.Where("(permissions.tenant_id = ? OR permissions.tenant_id IS NULL)", tenantID)
		query = query.Where("(roles.tenant_id = ? OR roles.tenant_id IS NULL)", tenantID)
		query = query.Where("(role_has_permissions.tenant_id = ? OR role_has_permissions.tenant_id IS NULL)", tenantID)
		query = query.Where("(model_has_roles.tenant_id = ? OR model_has_roles.tenant_id IS NULL)", tenantID)
	}

	if err := query.Find(&rolePerms).Error; err != nil {
		return nil, err
	}

	// Merge and deduplicate
	permMap := make(map[string]models.Permission)
	for _, p := range directPerms {
		permMap[p.Name] = p
	}
	for _, p := range rolePerms {
		permMap[p.Name] = p
	}

	var allPerms []models.Permission
	for _, p := range permMap {
		allPerms = append(allPerms, p)
	}

	return allPerms, nil
}
