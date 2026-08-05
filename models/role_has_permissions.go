package models

// RoleHasPermission represents the many-to-many relationship between roles and permissions
type RoleHasPermission struct {
	PermissionID uint    `gorm:"primaryKey" json:"permission_id"`
	RoleID       uint    `gorm:"primaryKey" json:"role_id"`
	TenantID     *string `gorm:"type:varchar(100);index:idx_role_permissions_tenant" json:"tenant_id,omitempty"` // Optional tenant
}

// TableName specifies the table name
func (RoleHasPermission) TableName() string {
	return "role_has_permissions"
}

// IsTenantScoped checks if the relationship is tenant-scoped
func (r *RoleHasPermission) IsTenantScoped() bool {
	return r.TenantID != nil && *r.TenantID != ""
}
