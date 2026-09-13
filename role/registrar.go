package role

import (
	"context"
	"errors"
	"fmt"

	"github.com/mwangaben/permission/config"
	"github.com/mwangaben/permission/storage"
	"github.com/mwangaben/permission/tenant"
)

// Registrar handles role registration with tenant support.
type Registrar struct {
	repo   storage.Repository
	config *config.Config
	tenant *tenant.Manager
}

// NewRegistrar creates a new role registrar.
func NewRegistrar(repo storage.Repository, config *config.Config, tenant *tenant.Manager) *Registrar {
	return &Registrar{repo: repo, config: config, tenant: tenant}
}

// Register creates a new role if it doesn't exist.
func (r *Registrar) Register(ctx context.Context, name string, guardName string) (*storage.Role, error) {
	if guardName == "" {
		guardName = r.config.DefaultGuard
	}

	tenantID := r.currentTenantID()

	existing, err := r.repo.FindRoleByName(ctx, name, guardName, tenantID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, storage.ErrRoleNotFound) {
		return nil, err
	}

	rl := &storage.Role{
		Name:      name,
		GuardName: guardName,
		TenantID:  tenantID,
	}
	if err := r.repo.CreateRole(ctx, rl); err != nil {
		return nil, fmt.Errorf("failed to create role: %w", err)
	}
	return rl, nil
}

// RegisterGlobal creates a global role (tenant_id = NULL).
func (r *Registrar) RegisterGlobal(ctx context.Context, name string, guardName string) (*storage.Role, error) {
	if guardName == "" {
		guardName = r.config.DefaultGuard
	}

	existing, err := r.repo.FindRoleByName(ctx, name, guardName, nil)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, storage.ErrRoleNotFound) {
		return nil, err
	}

	rl := &storage.Role{
		Name:      name,
		GuardName: guardName,
		TenantID:  nil,
	}
	if err := r.repo.CreateRole(ctx, rl); err != nil {
		return nil, fmt.Errorf("failed to create global role: %w", err)
	}
	return rl, nil
}

// FindByName finds a role by name, honoring tenant scoping.
func (r *Registrar) FindByName(ctx context.Context, name string, guardName string) (*storage.Role, error) {
	if guardName == "" {
		guardName = r.config.DefaultGuard
	}
	return r.repo.FindRoleByName(ctx, name, guardName, r.currentTenantID())
}

func (r *Registrar) currentTenantID() *string {
	if r.tenant == nil || !r.tenant.IsEnabled() {
		return nil
	}
	tid := r.tenant.GetTenantID()
	if tid == "" {
		return nil
	}
	return &tid
}
