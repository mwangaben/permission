package models

import (
	"time"

	"gorm.io/gorm"
)

// Role represents a role in the system
type Role struct {
	ID          string         `gorm:"primaryKey;type:varchar(100)" json:"id"`
	Name        string         `gorm:"type:varchar(255);uniqueIndex:idx_role_name_tenant;not null" json:"name"`
	DisplayName string         `gorm:"type:varchar(255)" json:"display_name"`
	Description string         `gorm:"type:text" json:"description"`
	TenantID    *string        `gorm:"type:varchar(100);index;uniqueIndex:idx_role_name_tenant" json:"tenant_id"`
	GuardName   string         `gorm:"type:varchar(100);default:web" json:"guard_name"`
	IsDefault   bool           `gorm:"default:false" json:"is_default"`
	Permissions []Permission   `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name
func (Role) TableName() string {
	return "roles"
}

// HasPermission checks if the role has a specific permission
func (r *Role) HasPermission(permissionName string) bool {
	for _, p := range r.Permissions {
		if p.Name == permissionName {
			return true
		}
	}
	return false
}

// HasAnyPermission checks if the role has any of the given permissions
func (r *Role) HasAnyPermission(permissionNames ...string) bool {
	for _, name := range permissionNames {
		if r.HasPermission(name) {
			return true
		}
	}
	return false
}

// HasAllPermissions checks if the role has all of the given permissions
func (r *Role) HasAllPermissions(permissionNames ...string) bool {
	for _, name := range permissionNames {
		if !r.HasPermission(name) {
			return false
		}
	}
	return true
}

// HasModelPermission checks if the role has any permission for a model
func (r *Role) HasModelPermission(model string, action string) bool {
	return r.HasPermission(model + "." + action)
}

// GetModelPermissions returns all permissions for a specific model
func (r *Role) GetModelPermissions(model string) []string {
	var permissions []string
	for _, p := range r.Permissions {
		if p.Model == model {
			permissions = append(permissions, p.Name)
		}
	}
	return permissions
}
