package models

import (
	"time"

	"gorm.io/gorm"
)

// Permission represents a permission in the system
// Format: {model}.{action} e.g., "user.view", "post.create", "report.delete"
type Permission struct {
	ID          string         `gorm:"primaryKey;type:varchar(100)" json:"id"`
	Name        string         `gorm:"type:varchar(255);uniqueIndex:idx_permission_name_tenant;not null" json:"name"`
	Model       string         `gorm:"type:varchar(100);index;not null" json:"model"` // e.g., "user", "post", "report"
	Action      string         `gorm:"type:varchar(50);index;not null" json:"action"` // e.g., "view", "create", "update", "delete"
	DisplayName string         `gorm:"type:varchar(255)" json:"display_name"`
	Description string         `gorm:"type:text" json:"description"`
	TenantID    *string        `gorm:"type:varchar(100);index;uniqueIndex:idx_permission_name_tenant" json:"tenant_id"`
	GuardName   string         `gorm:"type:varchar(100);default:web" json:"guard_name"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name
func (Permission) TableName() string {
	return "permissions"
}

// IsGlobal checks if the permission is global
func (p *Permission) IsGlobal() bool {
	return p.TenantID == nil || *p.TenantID == ""
}

// IsTenantSpecific checks if the permission is tenant-specific
func (p *Permission) IsTenantSpecific() bool {
	return !p.IsGlobal()
}

// String returns the permission name
func (p *Permission) String() string {
	return p.Name
}
