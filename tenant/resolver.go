package tenant

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mwangaben/permission/models"
	"gorm.io/gorm"
)

// Resolver handles tenant resolution and management
type Resolver struct {
	db *gorm.DB
}

// NewResolver creates a new tenant resolver
func NewResolver(db *gorm.DB) *Resolver {
	return &Resolver{db: db}
}

// ResolveTenant resolves a tenant from the request context
func (r *Resolver) ResolveTenant(ctx context.Context) (*models.Tenant, error) {
	tenantID, ok := ctx.Value("tenant_id").(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("tenant not found in context")
	}

	var tenant models.Tenant
	if err := r.db.Where("id = ? AND active = ?", tenantID, true).First(&tenant).Error; err != nil {
		return nil, fmt.Errorf("tenant not found: %w", err)
	}

	return &tenant, nil
}

// ResolveTenantByDomain resolves a tenant by domain
func (r *Resolver) ResolveTenantByDomain(domain string) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.Where("domain = ? AND active = ?", domain, true).First(&tenant).Error; err != nil {
		return nil, fmt.Errorf("tenant not found for domain: %w", err)
	}
	return &tenant, nil
}

// ResolveTenantBySlug resolves a tenant by slug
func (r *Resolver) ResolveTenantBySlug(slug string) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.Where("slug = ? AND active = ?", slug, true).First(&tenant).Error; err != nil {
		return nil, fmt.Errorf("tenant not found for slug: %w", err)
	}
	return &tenant, nil
}

// CreateTenant creates a new tenant
func (r *Resolver) CreateTenant(name, slug, domain, ownerID string) (*models.Tenant, error) {
	tenant := &models.Tenant{
		ID:      uuid.New().String(),
		Name:    name,
		Slug:    slug,
		Domain:  domain,
		OwnerID: ownerID,
		Active:  true,
	}

	if err := r.db.Create(tenant).Error; err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	return tenant, nil
}

// AddUserToTenant adds a user to a tenant
func (r *Resolver) AddUserToTenant(userID, tenantID string) error {
	// Check if already exists
	var count int64
	r.db.Table("tenant_user").
		Where("user_id = ? AND tenant_id = ?", userID, tenantID).
		Count(&count)

	if count > 0 {
		return nil
	}

	return r.db.Exec(
		"INSERT INTO tenant_user (user_id, tenant_id) VALUES (?, ?)",
		userID, tenantID,
	).Error
}

// RemoveUserFromTenant removes a user from a tenant
func (r *Resolver) RemoveUserFromTenant(userID, tenantID string) error {
	return r.db.Table("tenant_user").
		Where("user_id = ? AND tenant_id = ?", userID, tenantID).
		Delete(nil).Error
}

// GetTenantUsers returns all users belonging to a tenant
func (r *Resolver) GetTenantUsers(tenantID string) ([]string, error) {
	var userIDs []string
	if err := r.db.Table("tenant_user").
		Where("tenant_id = ?", tenantID).
		Pluck("user_id", &userIDs).Error; err != nil {
		return nil, fmt.Errorf("failed to get tenant users: %w", err)
	}
	return userIDs, nil
}

// GetUserTenants returns all tenants a user belongs to
func (r *Resolver) GetUserTenants(userID string) ([]models.Tenant, error) {
	var tenants []models.Tenant

	err := r.db.Model(&models.Tenant{}).
		Joins("INNER JOIN tenant_user ON tenant_user.tenant_id = tenants.id").
		Where("tenant_user.user_id = ? AND tenants.active = ?", userID, true).
		Find(&tenants).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get user tenants: %w", err)
	}

	return tenants, nil
}
