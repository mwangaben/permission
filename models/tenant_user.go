package models

import (
	"time"

	"gorm.io/gorm"
)

// TenantUser represents the many-to-many relationship between users and tenants
type TenantUser struct {
	UserID    string         `gorm:"type:varchar(100);primaryKey"`
	TenantID  string         `gorm:"type:varchar(100);primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name
func (TenantUser) TableName() string {
	return "tenant_user"
}
