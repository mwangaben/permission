package role

import (
	"github.com/mwangaben/permission/config"
	"github.com/mwangaben/permission/tenant"
	"gorm.io/gorm"
)

// Manager handles role operations with tenant support
type Manager struct {
	db          *gorm.DB
	config      *config.Config
	tenant      *tenant.Manager
	Registrar   *Registrar
	Assigner    *Assigner
	PermManager *PermissionManager
}

// NewManager creates a new role manager
func NewManager(db *gorm.DB, config *config.Config, tenant *tenant.Manager) *Manager {
	return &Manager{
		db:          db,
		config:      config,
		tenant:      tenant,
		Registrar:   NewRegistrar(db, config, tenant),
		Assigner:    NewAssigner(db),
		PermManager: NewPermissionManager(db),
	}
}

// WithTenant sets the tenant context
func (m *Manager) WithTenant(tenantID string) *Manager {
	// Create a new tenant manager with the new tenant ID
	newTenant := tenant.NewManager(m.tenant.IsEnabled()).WithTenant(tenantID)

	return &Manager{
		db:          m.db,
		config:      m.config,
		tenant:      newTenant,
		Registrar:   NewRegistrar(m.db, m.config, newTenant),
		Assigner:    m.Assigner,
		PermManager: m.PermManager,
	}
}
