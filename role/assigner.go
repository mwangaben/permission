package role

import (
	"github.com/mwangaben/permission/models"
	"gorm.io/gorm"
)

// Assigner handles role assignment
type Assigner struct {
	db *gorm.DB
}

// NewAssigner creates a new role assigner
func NewAssigner(db *gorm.DB) *Assigner {
	return &Assigner{db: db}
}

// Assign assigns a role to a user
func (a *Assigner) Assign(userID string, roleID string) error {
	var count int64
	a.db.Table("role_user").
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Count(&count)

	if count > 0 {
		return nil
	}

	return a.db.Exec(
		"INSERT INTO role_user (user_id, role_id) VALUES (?, ?)",
		userID, roleID,
	).Error
}

// AssignByName assigns a role to a user by role name
func (a *Assigner) AssignByName(userID string, roleName string, tenantID *string) error {
	var role models.Role
	query := a.db.Where("name = ?", roleName)
	if tenantID != nil && *tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	} else {
		query = query.Where("tenant_id IS NULL")
	}

	if err := query.First(&role).Error; err != nil {
		return err
	}

	return a.Assign(userID, role.ID)
}

// Remove removes a role from a user
func (a *Assigner) Remove(userID string, roleID string) error {
	return a.db.Table("role_user").
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Delete(nil).Error
}

// SyncRoles syncs roles for a user
func (a *Assigner) SyncRoles(userID string, roleIDs []string) error {
	tx := a.db.Begin()

	if err := tx.Table("role_user").
		Where("user_id = ?", userID).
		Delete(nil).Error; err != nil {
		tx.Rollback()
		return err
	}

	for _, roleID := range roleIDs {
		if err := tx.Exec(
			"INSERT INTO role_user (user_id, role_id) VALUES (?, ?)",
			userID, roleID,
		).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}
