package permission

import (
	"fmt"

	"github.com/mwangaben/permission/models"
	"gorm.io/gorm"
)

// DirectAssigner handles direct permission assignment to models (users)
type DirectAssigner struct {
	db *gorm.DB
}

// NewDirectAssigner creates a new direct permission assigner
func NewDirectAssigner(db *gorm.DB) *DirectAssigner {
	return &DirectAssigner{db: db}
}

// AssignPermissionToModel assigns a permission directly to a model (user)
func (a *DirectAssigner) AssignPermissionToModel(permissionID uint, modelType string, modelID uint) error {
	// Check if already assigned
	var count int64
	a.db.Model(&models.ModelHasPermission{}).
		Where("permission_id = ? AND model_type = ? AND model_id = ?", permissionID, modelType, modelID).
		Count(&count)

	if count > 0 {
		return nil // Already assigned
	}

	modelHasPermission := models.ModelHasPermission{
		PermissionID: permissionID,
		ModelType:    modelType,
		ModelID:      modelID,
	}

	return a.db.Create(&modelHasPermission).Error
}

// AssignPermissionToModelByName assigns a permission directly to a model by permission name
func (a *DirectAssigner) AssignPermissionToModelByName(permissionName, modelType string, modelID uint, guardName string) error {
	if guardName == "" {
		guardName = "web"
	}

	var permission models.Permission
	if err := a.db.Where("name = ? AND guard_name = ?", permissionName, guardName).First(&permission).Error; err != nil {
		return fmt.Errorf("permission not found: %w", err)
	}

	return a.AssignPermissionToModel(permission.ID, modelType, modelID)
}

// RemovePermissionFromModel removes a direct permission from a model
func (a *DirectAssigner) RemovePermissionFromModel(permissionID uint, modelType string, modelID uint) error {
	return a.db.Where("permission_id = ? AND model_type = ? AND model_id = ?", permissionID, modelType, modelID).
		Delete(&models.ModelHasPermission{}).Error
}

// GetDirectPermissionsForModel returns all direct permissions for a model
func (a *DirectAssigner) GetDirectPermissionsForModel(modelType string, modelID uint) ([]models.Permission, error) {
	var permissions []models.Permission

	err := a.db.Table("permissions").
		Joins("JOIN model_has_permissions ON model_has_permissions.permission_id = permissions.id").
		Where("model_has_permissions.model_type = ? AND model_has_permissions.model_id = ?", modelType, modelID).
		Find(&permissions).Error

	return permissions, err
}

// HasDirectPermission checks if a model has a specific direct permission
func (a *DirectAssigner) HasDirectPermission(modelType string, modelID uint, permissionName string, guardName string) (bool, error) {
	if guardName == "" {
		guardName = "web"
	}

	var count int64
	err := a.db.Table("model_has_permissions").
		Joins("JOIN permissions ON permissions.id = model_has_permissions.permission_id").
		Where("model_has_permissions.model_type = ? AND model_has_permissions.model_id = ?", modelType, modelID).
		Where("permissions.name = ? AND permissions.guard_name = ?", permissionName, guardName).
		Count(&count).Error

	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// SyncPermissionsForModel syncs direct permissions for a model (removes all and assigns new ones)
func (a *DirectAssigner) SyncPermissionsForModel(modelType string, modelID uint, permissionIDs []uint) error {
	tx := a.db.Begin()

	// Remove all existing direct permissions
	if err := tx.Where("model_type = ? AND model_id = ?", modelType, modelID).
		Delete(&models.ModelHasPermission{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Assign new permissions
	for _, permissionID := range permissionIDs {
		modelHasPermission := models.ModelHasPermission{
			PermissionID: permissionID,
			ModelType:    modelType,
			ModelID:      modelID,
		}
		if err := tx.Create(&modelHasPermission).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}
