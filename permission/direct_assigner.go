package permission

import (
	"context"
	"errors"
	"fmt"

	"github.com/mwangaben/permission/storage"
)

// DirectAssigner handles direct permission assignment to models (users).
type DirectAssigner struct {
	repo storage.Repository
}

func NewDirectAssigner(repo storage.Repository) *DirectAssigner {
	return &DirectAssigner{repo: repo}
}

// AssignPermissionToModel assigns a permission directly to a model.
func (a *DirectAssigner) AssignPermissionToModel(ctx context.Context, permissionID uint, modelType string, modelID uint, tenantID *string) error {
	return a.repo.AssignPermissionToModel(ctx, permissionID, modelType, modelID, tenantID)
}

// AssignPermissionToModelByName assigns a permission directly to a model by name.
func (a *DirectAssigner) AssignPermissionToModelByName(ctx context.Context, permissionName, modelType string, modelID uint, guardName string, tenantID *string) error {
	if guardName == "" {
		guardName = "web"
	}
	perm, err := a.repo.FindPermissionByName(ctx, permissionName, guardName, tenantID)
	if err != nil {
		if errors.Is(err, storage.ErrPermissionNotFound) {
			return fmt.Errorf("permission not found: %w", err)
		}
		return err
	}
	return a.repo.AssignPermissionToModel(ctx, perm.ID, modelType, modelID, tenantID)
}

func (a *DirectAssigner) RemovePermissionFromModel(ctx context.Context, permissionID uint, modelType string, modelID uint) error {
	return a.repo.RemovePermissionFromModel(ctx, permissionID, modelType, modelID)
}

func (a *DirectAssigner) GetDirectPermissionsForModel(ctx context.Context, modelType string, modelID uint, guardName string, tenantID *string) ([]*storage.Permission, error) {
	return a.repo.GetDirectPermissionsForModel(ctx, modelType, modelID, guardName, tenantID)
}

func (a *DirectAssigner) HasDirectPermission(ctx context.Context, modelType string, modelID uint, permissionName string, guardName string, tenantID *string) (bool, error) {
	if guardName == "" {
		guardName = "web"
	}
	count, err := a.repo.DirectPermissionCount(ctx, modelType, modelID, guardName, []string{permissionName}, tenantID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
