package models

// ModelHasRole represents the polymorphic relationship between models and roles.
//
// Surrogate ID added for Ent; composite uniqueness preserved via unique index.
type ModelHasRole struct {
	ID        uint    `gorm:"primaryKey" json:"id"`
	RoleID    uint    `gorm:"uniqueIndex:idx_model_roles_unique" json:"role_id"`
	ModelType string  `gorm:"type:varchar(255);uniqueIndex:idx_model_roles_unique" json:"model_type"`
	ModelID   uint    `gorm:"uniqueIndex:idx_model_roles_unique" json:"model_id"`
	TenantID  *string `gorm:"type:varchar(100);index:idx_model_roles_tenant" json:"tenant_id,omitempty"`
}

func (ModelHasRole) TableName() string {
	return "model_has_roles"
}

func (m *ModelHasRole) IsTenantScoped() bool {
	return m.TenantID != nil && *m.TenantID != ""
}
