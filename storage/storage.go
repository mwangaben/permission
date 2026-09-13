// Package storage defines a backend-agnostic persistence contract for the
// permission system. Two implementations are provided:
//
//   - storage/gormstore: GORM-backed, mirrors the models package
//   - storage/entstore:  Ent-backed, uses generated clients
//
// Consumers should never import GORM or Ent types directly when interacting
// with the permission system; instead, pass the raw DB handle (either
// *gorm.DB or *ent.Client) to NewManager, which auto-detects the backend.
package storage

import (
	"context"
	"errors"
)

// Repository is the persistence contract implemented by each backend.
//
// Every method takes a context.Context as its first argument so callers can
// propagate deadlines and cancellation. Every method returns storage-domain
// types (Permission, Role, etc.), never ORM-specific models.
//
// Tenant scoping rules (uniform across all methods):
//
//   - A nil tenantID or a pointer to an empty string means "global scope".
//     Reads return only rows with tenant_id IS NULL.
//   - A non-empty tenantID means "this tenant's scope, plus global rows".
//     Reads return rows where tenant_id = :tenantID OR tenant_id IS NULL.
//   - Writes always use the exact tenantID given (no NULL fallback); the
//     caller is responsible for deciding whether a write is tenant-scoped
//     or global.
type Repository interface {
	// ─── Permissions ────────────────────────────────────────────────────

	// CreatePermission inserts a new permission. Returns ErrPermissionExists
	// if a row with the same (name, guard_name, tenant_id) already exists.
	CreatePermission(ctx context.Context, p *Permission) error

	// FindPermissionByName returns a single permission matching the
	// (name, guard_name) pair, scoped by tenantID per the rules above.
	// Returns ErrPermissionNotFound if no such permission exists.
	FindPermissionByName(ctx context.Context, name, guardName string, tenantID *string) (*Permission, error)

	// FindPermissionByID returns a permission by primary key. NotFound is
	// returned for missing or soft-deleted rows.
	FindPermissionByID(ctx context.Context, id uint) (*Permission, error)

	// ─── Roles ──────────────────────────────────────────────────────────

	// CreateRole inserts a new role. Returns ErrRoleExists on duplicate
	// (name, guard_name, tenant_id).
	CreateRole(ctx context.Context, r *Role) error

	// FindRoleByName returns a single role matching (name, guard_name),
	// scoped by tenantID. Returns ErrRoleNotFound if missing.
	FindRoleByName(ctx context.Context, name, guardName string, tenantID *string) (*Role, error)

	// FindRoleByID returns a role by primary key.
	FindRoleByID(ctx context.Context, id uint) (*Role, error)

	// ─── Role ↔ Permission ──────────────────────────────────────────────

	// AssignPermissionToRole links a permission to a role. Under strict
	// tenant isolation:
	//   - If both are global, allowed.
	//   - If both belong to the same tenant, allowed.
	//   - If one is global and the other is tenant-scoped, allowed (global
	//     resources are shareable).
	//   - If both are tenant-scoped but different tenants, rejected with
	//     ErrCrossTenantAssignment.
	AssignPermissionToRole(ctx context.Context, permissionID, roleID uint) error

	// RemovePermissionFromRole is idempotent: removing a non-existent link
	// is not an error.
	RemovePermissionFromRole(ctx context.Context, permissionID, roleID uint) error

	// GetPermissionsForRole returns all permissions directly assigned to a
	// role. Excludes permissions inherited via other roles (there are none).
	GetPermissionsForRole(ctx context.Context, roleID uint) ([]*Permission, error)

	// RolePermissionExists reports whether a role has a permission matching
	// (name, guard) under the given tenant scope.
	//
	// Returns (false, nil) when the role simply doesn't have that permission.
	// Returns an error only for storage-layer failures.
	RolePermissionExists(ctx context.Context, roleID uint, permissionName, guardName string, tenantID *string) (bool, error)

	// ─── Model ↔ Role ───────────────────────────────────────────────────

	// AssignRoleToModel links a role to a polymorphic model (e.g. "user":42).
	// Same cross-tenant rules as AssignPermissionToRole.
	AssignRoleToModel(ctx context.Context, roleID uint, modelType string, modelID uint, tenantID *string) error

	// RemoveRoleFromModel is idempotent.
	RemoveRoleFromModel(ctx context.Context, roleID uint, modelType string, modelID uint) error

	// GetRolesForModel returns all roles assigned to the given model,
	// scoped by tenantID.
	GetRolesForModel(ctx context.Context, modelType string, modelID uint, tenantID *string) ([]*Role, error)

	// SyncRolesForModel replaces all roles for a model with the given set.
	// Implemented as a transaction: delete existing links, insert new ones.
	SyncRolesForModel(ctx context.Context, modelType string, modelID uint, roleIDs []uint, tenantID *string) error

	// ─── Model ↔ Permission (direct) ────────────────────────────────────

	// AssignPermissionToModel links a permission to a polymorphic model.
	// Same cross-tenant rules as AssignPermissionToRole.
	AssignPermissionToModel(ctx context.Context, permissionID uint, modelType string, modelID uint, tenantID *string) error

	// RemovePermissionFromModel is idempotent.
	RemovePermissionFromModel(ctx context.Context, permissionID uint, modelType string, modelID uint) error

	// GetDirectPermissionsForModel returns all permissions directly assigned
	// to a model (not via roles), filtered by guard.
	GetDirectPermissionsForModel(ctx context.Context, modelType string, modelID uint, guardName string, tenantID *string) ([]*Permission, error)

	// ─── Cross-table permission checks (used by Checker) ────────────────

	// DirectPermissionCount returns the number of direct permissions
	// matching modelType/modelID, guard, and any of the given names.
	// An empty names slice matches all permissions (used by callers that
	// want "any permission for this model").
	//
	// This replaces the first query in Checker.HasPermission and
	// Checker.HasAnyPermission.
	DirectPermissionCount(ctx context.Context, modelType string, modelID uint, guardName string, names []string, tenantID *string) (int, error)

	// RolePermissionCount returns the number of permissions reachable via
	// roles for the given model, filtered by guard and names.
	//
	// This replaces the second query in Checker.HasPermission and
	// Checker.HasAnyPermission.
	RolePermissionCount(ctx context.Context, modelType string, modelID uint, guardName string, names []string, tenantID *string) (int, error)

	// GetRoleDerivedPermissionsForModel returns all permissions a model has
	// through its roles, filtered by guard.
	//
	// This replaces the second query in Checker.GetAllPermissionsForModel.
	GetRoleDerivedPermissionsForModel(ctx context.Context, modelType string, modelID uint, guardName string, tenantID *string) ([]*Permission, error)

	// ─── Tenants ────────────────────────────────────────────────────────

	CreateTenant(ctx context.Context, t *Tenant) error
	FindTenantByID(ctx context.Context, id string) (*Tenant, error)
	FindTenantBySlug(ctx context.Context, slug string) (*Tenant, error)
	AssignUserToTenant(ctx context.Context, userID, tenantID string) error
	RemoveUserFromTenant(ctx context.Context, userID, tenantID string) error
	GetTenantsForUser(ctx context.Context, userID string) ([]*Tenant, error)

	// ─── Migration & Introspection ──────────────────────────────────────

	// AutoMigrate creates or updates all tables managed by the repository.
	// Both backends implement this as an idempotent operation.
	AutoMigrate(ctx context.Context) error

	// Name returns the backend identifier: "gorm" or "ent". Used for
	// logging and diagnostics.
	Name() string

	// RolesWithPermission returns all roles that have a permission matching
	// (name, guard) under the given tenant scope.
	RolesWithPermission(ctx context.Context, permissionName, guardName string, tenantID *string) ([]*Role, error)

	// SyncDirectPermissions atomically replaces the set of direct permissions
	// for a model.
	SyncDirectPermissions(ctx context.Context, modelType string, modelID uint, permissionIDs []uint, tenantID *string) error

	// SyncPermissionsForRole atomically replaces the set of permissions
	// assigned to a role. Tenant validation applies to each new link.
	SyncPermissionsForRole(ctx context.Context, roleID uint, permissionIDs []uint) error

	// RevokeAllPermissionsForRole removes every permission from a role,
	// leaving the role itself intact.
	RevokeAllPermissionsForRole(ctx context.Context, roleID uint) error
}

// ErrCrossTenantAssignment is returned when a caller attempts to link two
// tenant-scoped resources from different tenants. Global resources
// (tenant_id IS NULL) may be linked with anything.
var ErrCrossTenantAssignment = errors.New("storage: cannot link resources from different tenants")
