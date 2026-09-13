package role

import (
	"context"
	"errors"
	"fmt"

	"github.com/mwangaben/permission/storage"
)

// PermissionManager handles assigning permissions to roles.
type PermissionManager struct {
	repo storage.Repository
}

func NewPermissionManager(repo storage.Repository) *PermissionManager {
	return &PermissionManager{repo: repo}
}

// AssignPermissionToRole links a permission to a role.
//
// The repository enforces the strict cross-tenant rule.
func (pm *PermissionManager) AssignPermissionToRole(ctx context.Context, permissionID, roleID uint) error {
	return pm.repo.AssignPermissionToRole(ctx, permissionID, roleID)
}

// AssignPermissionToRoleByName looks up both entities by name and links them.
func (pm *PermissionManager) AssignPermissionToRoleByName(ctx context.Context, permissionName, roleName, guardName string, tenantID *string) error {
	if guardName == "" {
		guardName = "web"
	}
	perm, err := pm.repo.FindPermissionByName(ctx, permissionName, guardName, tenantID)
	if err != nil {
		if errors.Is(err, storage.ErrPermissionNotFound) {
			return fmt.Errorf("permission not found: %w", err)
		}
		return err
	}
	rl, err := pm.repo.FindRoleByName(ctx, roleName, guardName, tenantID)
	if err != nil {
		if errors.Is(err, storage.ErrRoleNotFound) {
			return fmt.Errorf("role not found: %w", err)
		}
		return err
	}
	return pm.repo.AssignPermissionToRole(ctx, perm.ID, rl.ID)
}

func (pm *PermissionManager) RemovePermissionFromRole(ctx context.Context, permissionID, roleID uint) error {
	return pm.repo.RemovePermissionFromRole(ctx, permissionID, roleID)
}

func (pm *PermissionManager) RemovePermissionFromRoleByName(ctx context.Context, permissionName, roleName, guardName string, tenantID *string) error {
	if guardName == "" {
		guardName = "web"
	}
	perm, err := pm.repo.FindPermissionByName(ctx, permissionName, guardName, tenantID)
	if err != nil {
		return err
	}
	rl, err := pm.repo.FindRoleByName(ctx, roleName, guardName, tenantID)
	if err != nil {
		return err
	}
	return pm.repo.RemovePermissionFromRole(ctx, perm.ID, rl.ID)
}

func (pm *PermissionManager) GetPermissionsForRole(ctx context.Context, roleID uint) ([]*storage.Permission, error) {
	return pm.repo.GetPermissionsForRole(ctx, roleID)
}

// HasPermissionForRole reports whether a role has a permission matching
// (name, guard) under the current tenant scope.
//
// This is a targeted existence check at the repository level; it does not
// load the role's full permission set.
func (pm *PermissionManager) HasPermissionForRole(ctx context.Context, roleID uint, permissionName, guardName string, tenantID *string) (bool, error) {
	if guardName == "" {
		guardName = "web"
	}
	return pm.repo.RolePermissionExists(ctx, roleID, permissionName, guardName, tenantID)
}

// SyncPermissionsForRole atomically replaces the set of permissions for a role.
func (pm *PermissionManager) SyncPermissionsForRole(ctx context.Context, roleID uint, permissionIDs []uint) error {
	return pm.repo.SyncPermissionsForRole(ctx, roleID, permissionIDs)
}

// RevokeAllPermissionsForRole removes every permission from a role.
func (pm *PermissionManager) RevokeAllPermissionsForRole(ctx context.Context, roleID uint) error {
	return pm.repo.RevokeAllPermissionsForRole(ctx, roleID)
}
