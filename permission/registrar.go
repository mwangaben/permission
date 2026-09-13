package permission

import (
	"context"
	"errors"
	"fmt"

	"github.com/mwangaben/permission/config"
	"github.com/mwangaben/permission/storage"
	"github.com/mwangaben/permission/tenant"
)

// Registrar handles permission registration with tenant support.
type Registrar struct {
	repo   storage.Repository
	config *config.Config
	tenant *tenant.Manager
}

func NewRegistrar(repo storage.Repository, config *config.Config, tenant *tenant.Manager) *Registrar {
	return &Registrar{repo: repo, config: config, tenant: tenant}
}

// Register creates a new permission if it doesn't exist.
func (r *Registrar) Register(ctx context.Context, name string, guardName string) (*storage.Permission, error) {
	if guardName == "" {
		guardName = r.config.DefaultGuard
	}

	tenantID := r.currentTenantID()

	// Idempotent lookup.
	existing, err := r.repo.FindPermissionByName(ctx, name, guardName, tenantID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, storage.ErrPermissionNotFound) {
		return nil, err
	}

	perm := &storage.Permission{
		Name:      name,
		GuardName: guardName,
		TenantID:  tenantID,
	}
	if err := r.repo.CreatePermission(ctx, perm); err != nil {
		return nil, fmt.Errorf("failed to create permission: %w", err)
	}
	return perm, nil
}

// RegisterGlobal creates a global permission (tenant_id = NULL).
func (r *Registrar) RegisterGlobal(ctx context.Context, name string, guardName string) (*storage.Permission, error) {
	if guardName == "" {
		guardName = r.config.DefaultGuard
	}

	existing, err := r.repo.FindPermissionByName(ctx, name, guardName, nil)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, storage.ErrPermissionNotFound) {
		return nil, err
	}

	perm := &storage.Permission{
		Name:      name,
		GuardName: guardName,
		TenantID:  nil,
	}
	if err := r.repo.CreatePermission(ctx, perm); err != nil {
		return nil, fmt.Errorf("failed to create global permission: %w", err)
	}
	return perm, nil
}

func (r *Registrar) RegisterMany(ctx context.Context, permissions []struct{ Name, GuardName string }) ([]*storage.Permission, error) {
	out := make([]*storage.Permission, 0, len(permissions))
	for _, p := range permissions {
		perm, err := r.Register(ctx, p.Name, p.GuardName)
		if err != nil {
			return nil, err
		}
		out = append(out, perm)
	}
	return out, nil
}

func (r *Registrar) FindByName(ctx context.Context, name string, guardName string) (*storage.Permission, error) {
	if guardName == "" {
		guardName = r.config.DefaultGuard
	}
	return r.repo.FindPermissionByName(ctx, name, guardName, r.currentTenantID())
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
