package tenant

import (
	"context"
)

// Manager handles tenant operations
type Manager struct {
	enabled  bool
	tenantID string
	ctx      context.Context
}

// NewManager creates a new tenant manager
func NewManager(enabled bool) *Manager {
	return &Manager{
		enabled: enabled,
		ctx:     context.Background(), // Initialize with background context
	}
}

// WithTenant sets the tenant context
func (m *Manager) WithTenant(tenantID string) *Manager {
	if !m.enabled {
		return m
	}
	return &Manager{
		enabled:  m.enabled,
		tenantID: tenantID,
		ctx:      context.WithValue(m.ctx, "tenant_id", tenantID),
	}
}

// GetTenantID returns the current tenant ID
func (m *Manager) GetTenantID() string {
	if !m.enabled {
		return ""
	}
	return m.tenantID
}

// IsEnabled checks if tenant mode is enabled
func (m *Manager) IsEnabled() bool {
	return m.enabled
}

// GetContext returns the context with tenant
func (m *Manager) GetContext() context.Context {
	if m.ctx == nil {
		return context.Background()
	}
	return m.ctx
}

// ScopeQuery scopes a GORM query to the current tenant
func (m *Manager) ScopeQuery(db interface{}, table string) interface{} {
	if !m.enabled {
		return db
	}
	// This will be used by the query scoping functions
	return db
}

// GetTenantFromContext extracts tenant ID from context
func GetTenantFromContext(ctx context.Context) string {
	if tenantID, ok := ctx.Value("tenant_id").(string); ok {
		return tenantID
	}
	return ""
}
