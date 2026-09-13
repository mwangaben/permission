package models

import (
	"time"

	"gorm.io/gorm"
)

type TenantUser struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    string         `gorm:"type:varchar(100);uniqueIndex:idx_tenant_user_unique" json:"user_id"`
	TenantID  string         `gorm:"type:varchar(100);uniqueIndex:idx_tenant_user_unique" json:"tenant_id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (TenantUser) TableName() string {
	return "tenant_user"
}
