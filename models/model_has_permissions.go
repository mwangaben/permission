package models

// ModelHasPermission represents the polymorphic relationship between models
// and permissions (direct assignment).
type ModelHasPermission struct {
	ID           uint    `gorm:"primaryKey" json:"id"`
	PermissionID uint    `gorm:"uniqueIndex:idx_model_permissions_unique" json:"permission_id"`
	ModelType    string  `gorm:"type:varchar(255);uniqueIndex:idx_model_permissions_unique" json:"model_type"`
	ModelID      uint    `gorm:"uniqueIndex:idx_model_permissions_unique" json:"model_id"`
	TenantID     *string `gorm:"type:varchar(100);index:idx_model_permissions_tenant" json:"tenant_id,omitempty"`
}

func (ModelHasPermission) TableName() string {
	return "model_has_permissions"
}

func (m *ModelHasPermission) IsTenantScoped() bool {
	return m.TenantID != nil && *m.TenantID != ""
}
