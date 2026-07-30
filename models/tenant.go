package models

import (
	"time"

	"gorm.io/gorm"
)

// Tenant represents a multi-tenant organization
type Tenant struct {
	ID        string         `gorm:"primaryKey;type:varchar(100)" json:"id"`
	Name      string         `gorm:"type:varchar(255);not null" json:"name"`
	Slug      string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"slug"`
	Domain    string         `gorm:"type:varchar(255);uniqueIndex" json:"domain"`
	Logo      string         `gorm:"type:varchar(255)" json:"logo"`
	Active    bool           `gorm:"default:true" json:"active"`
	OwnerID   string         `gorm:"type:varchar(100);index" json:"owner_id"`
	Config    string         `gorm:"type:json" json:"config"`
	Settings  string         `gorm:"type:json" json:"settings"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name
func (Tenant) TableName() string {
	return "tenants"
}

// IsActive checks if the tenant is active
func (t *Tenant) IsActive() bool {
	return t.Active
}

// IsOwner checks if a user is the owner of the tenant
func (t *Tenant) IsOwner(userID string) bool {
	return t.OwnerID == userID
}
