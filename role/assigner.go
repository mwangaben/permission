package role

import (
	"context"
	"errors"
	"fmt"

	"github.com/mwangaben/permission/storage"
)

// Assigner handles role assignment to models (users).
type Assigner struct {
	repo storage.Repository
}

// NewAssigner creates a new role assigner.
func NewAssigner(repo storage.Repository) *Assigner {
	return &Assigner{repo: repo}
}

// AssignRoleToModel assigns a role to a polymorphic model.
//
// The tenantID parameter determines the scope of the link. The repository
// enforces the strict cross-tenant rule.
func (a *Assigner) AssignRoleToModel(ctx context.Context, roleID uint, modelType string, modelID uint, tenantID *string) error {
	return a.repo.AssignRoleToModel(ctx, roleID, modelType, modelID, tenantID)
}

// AssignRoleToModelByName looks up the role by name and assigns it.
func (a *Assigner) AssignRoleToModelByName(ctx context.Context, roleName, modelType string, modelID uint, guardName string, tenantID *string) error {
	if guardName == "" {
		guardName = "web"
	}
	rl, err := a.repo.FindRoleByName(ctx, roleName, guardName, tenantID)
	if err != nil {
		if errors.Is(err, storage.ErrRoleNotFound) {
			return fmt.Errorf("role not found: %w", err)
		}
		return err
	}
	return a.repo.AssignRoleToModel(ctx, rl.ID, modelType, modelID, tenantID)
}

// RemoveRoleFromModel removes a role from a polymorphic model. Idempotent.
func (a *Assigner) RemoveRoleFromModel(ctx context.Context, roleID uint, modelType string, modelID uint) error {
	return a.repo.RemoveRoleFromModel(ctx, roleID, modelType, modelID)
}

// GetRolesForModel returns all roles for a model.
func (a *Assigner) GetRolesForModel(ctx context.Context, modelType string, modelID uint, tenantID *string) ([]*storage.Role, error) {
	return a.repo.GetRolesForModel(ctx, modelType, modelID, tenantID)
}

// SyncRolesForModel replaces all roles for a model.
func (a *Assigner) SyncRolesForModel(ctx context.Context, modelType string, modelID uint, roleIDs []uint, tenantID *string) error {
	return a.repo.SyncRolesForModel(ctx, modelType, modelID, roleIDs, tenantID)
}

// ─── Deprecated aliases kept for backward compatibility ─────────────────

func (a *Assigner) Assign(ctx context.Context, userID uint, roleID uint, tenantID *string) error {
	return a.AssignRoleToModel(ctx, roleID, "user", userID, tenantID)
}

func (a *Assigner) AssignByName(ctx context.Context, userID uint, roleName string, tenantID *string) error {
	return a.AssignRoleToModelByName(ctx, roleName, "user", userID, "web", tenantID)
}

func (a *Assigner) Remove(ctx context.Context, userID uint, roleID uint) error {
	return a.RemoveRoleFromModel(ctx, roleID, "user", userID)
}

func (a *Assigner) SyncRoles(ctx context.Context, userID uint, roleIDs []uint, tenantID *string) error {
	return a.SyncRolesForModel(ctx, "user", userID, roleIDs, tenantID)
}
