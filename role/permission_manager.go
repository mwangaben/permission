package role

import (
	"fmt"

	"github.com/mwangaben/permission/models"
	"gorm.io/gorm"
)

// PermissionManager handles assigning permissions to roles
type PermissionManager struct {
	db *gorm.DB
}

// NewPermissionManager creates a new permission manager
func NewPermissionManager(db *gorm.DB) *PermissionManager {
	return &PermissionManager{db: db}
}

// AssignPermissionToRole assigns a permission to a role
func (pm *PermissionManager) AssignPermissionToRole(permissionID, roleID uint) error {
	// Check if already assigned
	var count int64
	pm.db.Model(&models.RoleHasPermission{}).
		Where("permission_id = ? AND role_id = ?", permissionID, roleID).
		Count(&count)

	if count > 0 {
		return nil // Already assigned
	}

	roleHasPermission := models.RoleHasPermission{
		PermissionID: permissionID,
		RoleID:       roleID,
	}

	return pm.db.Create(&roleHasPermission).Error
}

// AssignPermissionToRoleByName assigns a permission to a role by name
func (pm *PermissionManager) AssignPermissionToRoleByName(permissionName, roleName, guardName string) error {
	if guardName == "" {
		guardName = "web"
	}

	var permission models.Permission
	if err := pm.db.Where("name = ? AND guard_name = ?", permissionName, guardName).First(&permission).Error; err != nil {
		return fmt.Errorf("permission not found: %w", err)
	}

	var role models.Role
	if err := pm.db.Where("name = ? AND guard_name = ?", roleName, guardName).First(&role).Error; err != nil {
		return fmt.Errorf("role not found: %w", err)
	}

	return pm.AssignPermissionToRole(permission.ID, role.ID)
}

// RemovePermissionFromRole removes a permission from a role
func (pm *PermissionManager) RemovePermissionFromRole(permissionID, roleID uint) error {
	return pm.db.Where("permission_id = ? AND role_id = ?", permissionID, roleID).
		Delete(&models.RoleHasPermission{}).Error
}

// RemovePermissionFromRoleByName removes a permission from a role by name (revoke)
func (pm *PermissionManager) RemovePermissionFromRoleByName(permissionName, roleName, guardName string) error {
	if guardName == "" {
		guardName = "web"
	}

	var permission models.Permission
	if err := pm.db.Where("name = ? AND guard_name = ?", permissionName, guardName).First(&permission).Error; err != nil {
		return fmt.Errorf("permission not found: %w", err)
	}

	var role models.Role
	if err := pm.db.Where("name = ? AND guard_name = ?", roleName, guardName).First(&role).Error; err != nil {
		return fmt.Errorf("role not found: %w", err)
	}

	return pm.RemovePermissionFromRole(permission.ID, role.ID)
}

// GetPermissionsForRole returns all permissions for a role
func (pm *PermissionManager) GetPermissionsForRole(roleID uint) ([]models.Permission, error) {
	var permissions []models.Permission

	err := pm.db.Table("permissions").
		Joins("JOIN role_has_permissions ON role_has_permissions.permission_id = permissions.id").
		Where("role_has_permissions.role_id = ?", roleID).
		Find(&permissions).Error

	return permissions, err
}

// HasPermission checks if a role has a specific permission
func (pm *PermissionManager) HasPermission(roleID uint, permissionName string) (bool, error) {
	var count int64
	err := pm.db.Table("role_has_permissions").
		Joins("JOIN permissions ON permissions.id = role_has_permissions.permission_id").
		Where("role_has_permissions.role_id = ? AND permissions.name = ?", roleID, permissionName).
		Count(&count).Error

	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// SyncPermissionsForRole syncs permissions for a role
func (pm *PermissionManager) SyncPermissionsForRole(roleID uint, permissionIDs []uint) error {
	tx := pm.db.Begin()

	// Remove all existing permissions
	if err := tx.Where("role_id = ?", roleID).Delete(&models.RoleHasPermission{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Assign new permissions
	for _, permissionID := range permissionIDs {
		roleHasPermission := models.RoleHasPermission{
			PermissionID: permissionID,
			RoleID:       roleID,
		}
		if err := tx.Create(&roleHasPermission).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}
