package models

import (
	"time"

	"gorm.io/gorm"
)

// Permission represents a permission in the system
type Permission struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"type:varchar(255);uniqueIndex:idx_permissions_name_guard_tenant;not null" json:"name"`
	GuardName string         `gorm:"type:varchar(100);default:web;uniqueIndex:idx_permissions_name_guard_tenant" json:"guard_name"`
	TenantID  *string        `gorm:"type:varchar(100);uniqueIndex:idx_permissions_name_guard_tenant;index:idx_permissions_tenant" json:"tenant_id,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name
func (Permission) TableName() string {
	return "permissions"
}

// IsTenantScoped checks if the permission is tenant-scoped
func (p *Permission) IsTenantScoped() bool {
	return p.TenantID != nil && *p.TenantID != ""
}

// ScopeTenant scopes the query to a specific tenant
func (p *Permission) ScopeTenant(db *gorm.DB, tenantID string) *gorm.DB {
	if tenantID == "" {
		return db.Where("tenant_id IS NULL")
	}
	return db.Where("tenant_id = ? OR tenant_id IS NULL", tenantID)
}
