package permission

import (
	"context"
	"fmt"

	"github.com/mwangaben/permission/config"
	"github.com/mwangaben/permission/storage"
	"github.com/mwangaben/permission/tenant"
)

// Checker handles permission checking with tenant support.
//
// Checker is storage-agnostic: it talks to storage.Repository. The concrete
// backend (GORM or Ent) is chosen by whoever constructed the repository.
type Checker struct {
	repo   storage.Repository
	config *config.Config
	tenant *tenant.Manager
}

// NewChecker creates a new permission checker backed by a storage.Repository.
func NewChecker(repo storage.Repository, config *config.Config, tenant *tenant.Manager) *Checker {
	return &Checker{
		repo:   repo,
		config: config,
		tenant: tenant,
	}
}

// HasPermission checks if a model has a specific permission.
func (c *Checker) HasPermission(ctx context.Context, modelType string, modelID uint, permissionName string, guardName string) (bool, error) {
	if guardName == "" {
		guardName = c.config.DefaultGuard
	}

	tenantID := c.currentTenantID()

	direct, err := c.repo.DirectPermissionCount(ctx, modelType, modelID, guardName, []string{permissionName}, tenantID)
	if err != nil {
		return false, err
	}
	if direct > 0 {
		return true, nil
	}

	viaRole, err := c.repo.RolePermissionCount(ctx, modelType, modelID, guardName, []string{permissionName}, tenantID)
	if err != nil {
		return false, err
	}
	return viaRole > 0, nil
}

// HasAnyPermission checks if a model has any of the given permissions.
func (c *Checker) HasAnyPermission(ctx context.Context, modelType string, modelID uint, guardName string, permissionNames ...string) (bool, error) {
	if guardName == "" {
		guardName = c.config.DefaultGuard
	}
	if len(permissionNames) == 0 {
		return false, nil
	}

	tenantID := c.currentTenantID()

	direct, err := c.repo.DirectPermissionCount(ctx, modelType, modelID, guardName, permissionNames, tenantID)
	if err != nil {
		return false, err
	}
	if direct > 0 {
		return true, nil
	}

	viaRole, err := c.repo.RolePermissionCount(ctx, modelType, modelID, guardName, permissionNames, tenantID)
	if err != nil {
		return false, err
	}
	return viaRole > 0, nil
}

// GetAllPermissionsForModel returns all permissions for a model, deduplicated
// by (name, guard_name, tenant_id).
func (c *Checker) GetAllPermissionsForModel(ctx context.Context, modelType string, modelID uint, guardName string) ([]*storage.Permission, error) {
	if guardName == "" {
		guardName = c.config.DefaultGuard
	}

	tenantID := c.currentTenantID()

	direct, err := c.repo.GetDirectPermissionsForModel(ctx, modelType, modelID, guardName, tenantID)
	if err != nil {
		return nil, err
	}

	viaRole, err := c.repo.GetRoleDerivedPermissionsForModel(ctx, modelType, modelID, guardName, tenantID)
	if err != nil {
		return nil, err
	}

	// Deduplicate on (name, guard_name, tenant_id). Previously this code
	// keyed only on name, which collapsed permissions from different guards
	// or tenants into one.
	type key struct {
		Name      string
		GuardName string
		TenantID  string
	}
	seen := make(map[key]*storage.Permission, len(direct)+len(viaRole))

	tenantKey := func(p *storage.Permission) string {
		if p.TenantID == nil {
			return ""
		}
		return *p.TenantID
	}

	for _, p := range direct {
		k := key{p.Name, p.GuardName, tenantKey(p)}
		if _, exists := seen[k]; !exists {
			seen[k] = p
		}
	}
	for _, p := range viaRole {
		k := key{p.Name, p.GuardName, tenantKey(p)}
		if _, exists := seen[k]; !exists {
			seen[k] = p
		}
	}

	out := make([]*storage.Permission, 0, len(seen))
	for _, p := range seen {
		out = append(out, p)
	}
	return out, nil
}

// currentTenantID returns the tenant ID pointer to pass to the repository,
// or nil when tenant mode is disabled.
func (c *Checker) currentTenantID() *string {
	if c.tenant == nil || !c.tenant.IsEnabled() {
		return nil
	}
	tid := c.tenant.GetTenantID()
	if tid == "" {
		return nil
	}
	return &tid
}

var _ = fmt.Sprintf // keep fmt import if unused elsewhere
