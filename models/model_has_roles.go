package models

// ModelHasRole represents the polymorphic relationship between models and roles
type ModelHasRole struct {
	RoleID    uint    `gorm:"primaryKey" json:"role_id"`
	ModelType string  `gorm:"type:varchar(255);primaryKey" json:"model_type"`
	ModelID   uint    `gorm:"primaryKey" json:"model_id"`
	TenantID  *string `gorm:"type:varchar(100);index:idx_model_roles_tenant" json:"tenant_id,omitempty"` // Optional tenant
}

// TableName specifies the table name
func (ModelHasRole) TableName() string {
	return "model_has_roles"
}

// IsTenantScoped checks if the relationship is tenant-scoped
func (m *ModelHasRole) IsTenantScoped() bool {
	return m.TenantID != nil && *m.TenantID != ""
}
