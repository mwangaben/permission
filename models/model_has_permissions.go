package models

// ModelHasPermission represents the polymorphic relationship between models and permissions
type ModelHasPermission struct {
	PermissionID uint    `gorm:"primaryKey" json:"permission_id"`
	ModelType    string  `gorm:"type:varchar(255);primaryKey" json:"model_type"`
	ModelID      uint    `gorm:"primaryKey" json:"model_id"`
	TenantID     *string `gorm:"type:varchar(100);index:idx_model_permissions_tenant" json:"tenant_id,omitempty"` // Optional tenant
}

// TableName specifies the table name
func (ModelHasPermission) TableName() string {
	return "model_has_permissions"
}

// IsTenantScoped checks if the relationship is tenant-scoped
func (m *ModelHasPermission) IsTenantScoped() bool {
	return m.TenantID != nil && *m.TenantID != ""
}
