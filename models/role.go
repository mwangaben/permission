package models

import (
	"time"

	"gorm.io/gorm"
)

// Role represents a role in the system
type Role struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"type:varchar(255);uniqueIndex:idx_roles_name_guard_tenant;not null" json:"name"`
	GuardName   string         `gorm:"type:varchar(100);default:web;uniqueIndex:idx_roles_name_guard_tenant" json:"guard_name"`
	TenantID    *string        `gorm:"type:varchar(100);uniqueIndex:idx_roles_name_guard_tenant;index:idx_roles_tenant" json:"tenant_id,omitempty"`
	Permissions []Permission   `gorm:"many2many:role_has_permissions;" json:"permissions,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name
func (Role) TableName() string {
	return "roles"
}

// IsTenantScoped checks if the role is tenant-scoped
func (r *Role) IsTenantScoped() bool {
	return r.TenantID != nil && *r.TenantID != ""
}

// ScopeTenant scopes the query to a specific tenant
func (r *Role) ScopeTenant(db *gorm.DB, tenantID string) *gorm.DB {
	if tenantID == "" {
		return db.Where("tenant_id IS NULL")
	}
	return db.Where("tenant_id = ? OR tenant_id IS NULL", tenantID)
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
