package permission

import (
	"context"

	"github.com/mwangaben/permission/models"
	"gorm.io/gorm"
)

// Checker handles permission checking
type Checker struct {
	db *gorm.DB
}

// NewChecker creates a new permission checker
func NewChecker(db *gorm.DB) *Checker {
	return &Checker{db: db}
}

// HasPermission checks if a user has a specific permission
func (c *Checker) HasPermission(ctx context.Context, userID string, permissionName string, tenantID *string) (bool, error) {
	var count int64

	// Use Unscoped to avoid soft delete on join table
	query := c.db.Unscoped().Table("role_user").
		Joins("JOIN roles ON roles.id = role_user.role_id").
		Joins("JOIN role_permissions ON role_permissions.role_id = roles.id").
		Joins("JOIN permissions ON permissions.id = role_permissions.permission_id").
		Where("role_user.user_id = ? AND permissions.name = ? AND roles.deleted_at IS NULL", userID, permissionName)

	if tenantID != nil && *tenantID != "" {
		query = query.Where("(roles.tenant_id = ? OR roles.tenant_id IS NULL)", *tenantID)
		query = query.Where("(permissions.tenant_id = ? OR permissions.tenant_id IS NULL)", *tenantID)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

// HasModelPermission checks if a user has a specific model permission
func (c *Checker) HasModelPermission(ctx context.Context, userID, model, action string, tenantID *string) (bool, error) {
	return c.HasPermission(ctx, userID, model+"."+action, tenantID)
}

// HasAnyPermission checks if a user has any of the given permissions
func (c *Checker) HasAnyPermission(ctx context.Context, userID string, tenantID *string, permissionNames ...string) (bool, error) {
	if len(permissionNames) == 0 {
		return false, nil
	}

	var count int64

	// Use Unscoped to avoid soft delete on join table
	query := c.db.Unscoped().Table("role_user").
		Joins("JOIN roles ON roles.id = role_user.role_id").
		Joins("JOIN role_permissions ON role_permissions.role_id = roles.id").
		Joins("JOIN permissions ON permissions.id = role_permissions.permission_id").
		Where("role_user.user_id = ? AND permissions.name IN (?) AND roles.deleted_at IS NULL", userID, permissionNames)

	if tenantID != nil && *tenantID != "" {
		query = query.Where("(roles.tenant_id = ? OR roles.tenant_id IS NULL)", *tenantID)
		query = query.Where("(permissions.tenant_id = ? OR permissions.tenant_id IS NULL)", *tenantID)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

// HasAllPermissions checks if a user has all of the given permissions
func (c *Checker) HasAllPermissions(ctx context.Context, userID string, tenantID *string, permissionNames ...string) (bool, error) {
	if len(permissionNames) == 0 {
		return true, nil
	}

	for _, perm := range permissionNames {
		has, err := c.HasPermission(ctx, userID, perm, tenantID)
		if err != nil || !has {
			return false, err
		}
	}

	return true, nil
}

// GetUserPermissions returns all permissions for a user
func (c *Checker) GetUserPermissions(ctx context.Context, userID string, tenantID *string) ([]models.Permission, error) {
	var permissions []models.Permission

	// Use Unscoped to avoid soft delete on join table
	query := c.db.Unscoped().Table("role_user").
		Select("permissions.*").
		Joins("JOIN roles ON roles.id = role_user.role_id").
		Joins("JOIN role_permissions ON role_permissions.role_id = roles.id").
		Joins("JOIN permissions ON permissions.id = role_permissions.permission_id").
		Where("role_user.user_id = ? AND roles.deleted_at IS NULL", userID)

	if tenantID != nil && *tenantID != "" {
		query = query.Where("(roles.tenant_id = ? OR roles.tenant_id IS NULL)", *tenantID)
		query = query.Where("(permissions.tenant_id = ? OR permissions.tenant_id IS NULL)", *tenantID)
	}

	if err := query.Find(&permissions).Error; err != nil {
		return nil, err
	}

	return permissions, nil
}

// GetUserModelPermissions returns all permissions for a specific model
func (c *Checker) GetUserModelPermissions(ctx context.Context, userID, model string, tenantID *string) ([]string, error) {
	var permissions []string

	// Use Unscoped to avoid soft delete on join table
	query := c.db.Unscoped().Table("role_user").
		Select("permissions.name").
		Joins("JOIN roles ON roles.id = role_user.role_id").
		Joins("JOIN role_permissions ON role_permissions.role_id = roles.id").
		Joins("JOIN permissions ON permissions.id = role_permissions.permission_id").
		Where("role_user.user_id = ? AND permissions.model = ? AND roles.deleted_at IS NULL", userID, model)

	if tenantID != nil && *tenantID != "" {
		query = query.Where("(roles.tenant_id = ? OR roles.tenant_id IS NULL)", *tenantID)
		query = query.Where("(permissions.tenant_id = ? OR permissions.tenant_id IS NULL)", *tenantID)
	}

	if err := query.Find(&permissions).Error; err != nil {
		return nil, err
	}

	return permissions, nil
}

// GetUserPermissionNames returns all permission names for a user
func (c *Checker) GetUserPermissionNames(ctx context.Context, userID string, tenantID *string) ([]string, error) {
	permissions, err := c.GetUserPermissions(ctx, userID, tenantID)
	if err != nil {
		return nil, err
	}

	names := make([]string, len(permissions))
	for i, p := range permissions {
		names[i] = p.Name
	}
	return names, nil
}
