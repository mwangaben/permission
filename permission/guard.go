package permission

import (
	"context"
	"fmt"
)

// Guard provides a high-level API for permission checking.
type Guard struct {
	checker *Checker
}

func NewGuard(checker *Checker) *Guard {
	return &Guard{checker: checker}
}

func (g *Guard) Allows(ctx context.Context, modelType string, modelID uint, permission string, guardName string) (bool, error) {
	return g.checker.HasPermission(ctx, modelType, modelID, permission, guardName)
}

func (g *Guard) Denies(ctx context.Context, modelType string, modelID uint, permission string, guardName string) (bool, error) {
	allowed, err := g.Allows(ctx, modelType, modelID, permission, guardName)
	if err != nil {
		return false, err
	}
	return !allowed, nil
}

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

func (g *Guard) AuthorizeAll(ctx context.Context, modelType string, modelID uint, guardName string, permissions ...string) error {
	for _, perm := range permissions {
		if err := g.Authorize(ctx, modelType, modelID, perm, guardName); err != nil {
			return err
		}
	}
	return nil
}

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
