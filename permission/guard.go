package permission

import (
	"context"
	"fmt"
)

// Guard provides a high-level API for permission checking
type Guard struct {
	checker *Checker
}

// NewGuard creates a new permission guard
func NewGuard(checker *Checker) *Guard {
	return &Guard{checker: checker}
}

// Allows checks if a model can perform an action
func (g *Guard) Allows(ctx context.Context, modelType string, modelID uint, permission string, guardName string) (bool, error) {
	return g.checker.HasPermission(modelType, modelID, permission, guardName)
}

// Denies checks if a model cannot perform an action
func (g *Guard) Denies(ctx context.Context, modelType string, modelID uint, permission string, guardName string) (bool, error) {
	allowed, err := g.Allows(ctx, modelType, modelID, permission, guardName)
	return !allowed, err
}

// Authorize checks if a model can perform an action and returns an error if not
func (g *Guard) Authorize(ctx context.Context, modelType string, modelID uint, permission string, guardName string) error {
	allowed, err := g.Allows(ctx, modelType, modelID, permission, guardName)
	if err != nil {
		return err
	}
	if !allowed {
		return fmt.Errorf("unauthorized: missing %s permission", permission)
	}
	return nil
}

// AuthorizeAll checks if a model has all permissions
func (g *Guard) AuthorizeAll(ctx context.Context, modelType string, modelID uint, guardName string, permissions ...string) error {
	for _, perm := range permissions {
		if err := g.Authorize(ctx, modelType, modelID, perm, guardName); err != nil {
			return err
		}
	}
	return nil
}

// AuthorizeAny checks if a model has any of the permissions
func (g *Guard) AuthorizeAny(ctx context.Context, modelType string, modelID uint, guardName string, permissions ...string) error {
	for _, perm := range permissions {
		allowed, err := g.Allows(ctx, modelType, modelID, perm, guardName)
		if err != nil {
			return err
		}
		if allowed {
			return nil
		}
	}
	return fmt.Errorf("unauthorized: missing any of the required permissions")
}
