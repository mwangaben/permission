package storage

import "time"

// Permission is a stored permission record, backend-agnostic.
//
// TenantID is nil for global permissions. IsTenantScoped reports whether
// the permission is bound to a specific tenant.
type Permission struct {
	ID        uint
	Name      string
	GuardName string
	TenantID  *string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time // nil when not soft-deleted
}

func (p *Permission) IsTenantScoped() bool {
	return p.TenantID != nil && *p.TenantID != ""
}

func (p *Permission) IsDeleted() bool {
	return p.DeletedAt != nil
}

// Role is a stored role record, backend-agnostic.
type Role struct {
	ID        uint
	Name      string
	GuardName string
	TenantID  *string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

func (r *Role) IsTenantScoped() bool {
	return r.TenantID != nil && *r.TenantID != ""
}

func (r *Role) IsDeleted() bool {
	return r.DeletedAt != nil
}

// RoleHasPermission is the pivot linking roles to permissions.
type RoleHasPermission struct {
	ID           uint // surrogate PK
	PermissionID uint
	RoleID       uint
	TenantID     *string
}

// ModelHasRole is the polymorphic pivot linking roles to arbitrary models.
type ModelHasRole struct {
	ID        uint // surrogate PK
	RoleID    uint
	ModelType string
	ModelID   uint
	TenantID  *string
}

// ModelHasPermission is the polymorphic pivot linking permissions directly
// to arbitrary models.
type ModelHasPermission struct {
	ID           uint // surrogate PK
	PermissionID uint
	ModelType    string
	ModelID      uint
	TenantID     *string
}

// Tenant is a multi-tenant organization.
type Tenant struct {
	ID        string
	Name      string
	Slug      string
	Domain    string
	Logo      string
	Active    bool
	OwnerID   string
	Config    string // JSON blob
	Settings  string // JSON blob
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

func (t *Tenant) IsActive() bool             { return t.Active }
func (t *Tenant) IsOwner(userID string) bool { return t.OwnerID == userID }

// TenantUser links users to tenants (many-to-many).
type TenantUser struct {
	ID        uint // surrogate PK
	UserID    string
	TenantID  string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// ─── Filter types (kept small; expand only when needed) ─────────────────

// TenantScope normalizes a nullable tenant ID pointer into a tri-state:
//
//   - ScopeGlobal    — read only global rows (tenant_id IS NULL)
//   - ScopeTenant    — read tenant rows + global rows
//
// Encoded as a helper rather than as an enum because the two backends
// already implement this via inline WHERE clauses.
func TenantScope(tenantID *string) (string, bool) {
	if tenantID == nil || *tenantID == "" {
		return "", false // global scope
	}
	return *tenantID, true // tenant scope
}
