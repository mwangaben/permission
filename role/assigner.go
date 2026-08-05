package role

import (
	"fmt"

	"github.com/mwangaben/permission/models"
	"gorm.io/gorm"
)

// Assigner handles role assignment to models (users)
type Assigner struct {
	db *gorm.DB
}

// NewAssigner creates a new role assigner
func NewAssigner(db *gorm.DB) *Assigner {
	return &Assigner{db: db}
}

// AssignRoleToModel assigns a role to a model (user) using polymorphic relationship
func (a *Assigner) AssignRoleToModel(roleID uint, modelType string, modelID uint) error {
	// Check if already assigned
	var count int64
	a.db.Model(&models.ModelHasRole{}).
		Where("role_id = ? AND model_type = ? AND model_id = ?", roleID, modelType, modelID).
		Count(&count)

	if count > 0 {
		return nil // Already assigned
	}

	modelHasRole := models.ModelHasRole{
		RoleID:    roleID,
		ModelType: modelType,
		ModelID:   modelID,
	}

	return a.db.Create(&modelHasRole).Error
}

// AssignRoleToModelByName assigns a role to a model by role name
func (a *Assigner) AssignRoleToModelByName(roleName, modelType string, modelID uint, guardName string) error {
	if guardName == "" {
		guardName = "web"
	}

	var role models.Role
	if err := a.db.Where("name = ? AND guard_name = ?", roleName, guardName).First(&role).Error; err != nil {
		return fmt.Errorf("role not found: %w", err)
	}

	return a.AssignRoleToModel(role.ID, modelType, modelID)
}

// RemoveRoleFromModel removes a role from a model
func (a *Assigner) RemoveRoleFromModel(roleID uint, modelType string, modelID uint) error {
	return a.db.Where("role_id = ? AND model_type = ? AND model_id = ?", roleID, modelType, modelID).
		Delete(&models.ModelHasRole{}).Error
}

// GetRolesForModel returns all roles for a model
func (a *Assigner) GetRolesForModel(modelType string, modelID uint) ([]models.Role, error) {
	var roles []models.Role

	err := a.db.Table("roles").
		Joins("JOIN model_has_roles ON model_has_roles.role_id = roles.id").
		Where("model_has_roles.model_type = ? AND model_has_roles.model_id = ?", modelType, modelID).
		Find(&roles).Error

	return roles, err
}

// SyncRolesForModel syncs roles for a model (removes all and assigns new ones)
func (a *Assigner) SyncRolesForModel(modelType string, modelID uint, roleIDs []uint) error {
	tx := a.db.Begin()

	// Remove all existing roles
	if err := tx.Where("model_type = ? AND model_id = ?", modelType, modelID).
		Delete(&models.ModelHasRole{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Assign new roles
	for _, roleID := range roleIDs {
		modelHasRole := models.ModelHasRole{
			RoleID:    roleID,
			ModelType: modelType,
			ModelID:   modelID,
		}
		if err := tx.Create(&modelHasRole).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// ============ DEPRECATED METHODS (for backward compatibility) ============

// Assign assigns a role to a user (DEPRECATED - use AssignRoleToModel)
// This is kept for backward compatibility but uses the old role_user table
func (a *Assigner) Assign(userID uint, roleID uint) error {
	return a.AssignRoleToModel(roleID, "user", userID)
}

// AssignByName assigns a role to a user by role name (DEPRECATED - use AssignRoleToModelByName)
func (a *Assigner) AssignByName(userID uint, roleName string, tenantID *string) error {
	// Convert tenantID to string pointer if needed
	var tenantIDPtr *string
	if tenantID != nil {
		tenantIDPtr = tenantID
	}

	var role models.Role
	query := a.db.Where("name = ?", roleName)
	if tenantIDPtr != nil && *tenantIDPtr != "" {
		query = query.Where("tenant_id = ?", tenantIDPtr)
	} else {
		query = query.Where("tenant_id IS NULL")
	}

	if err := query.First(&role).Error; err != nil {
		return err
	}

	return a.AssignRoleToModel(role.ID, "user", userID)
}

// Remove removes a role from a user (DEPRECATED - use RemoveRoleFromModel)
func (a *Assigner) Remove(userID uint, roleID uint) error {
	return a.RemoveRoleFromModel(roleID, "user", userID)
}

// SyncRoles syncs roles for a user (DEPRECATED - use SyncRolesForModel)
func (a *Assigner) SyncRoles(userID uint, roleIDs []uint) error {
	return a.SyncRolesForModel("user", userID, roleIDs)
}
