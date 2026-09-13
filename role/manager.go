package role

import (
	"github.com/mwangaben/permission/config"
	"github.com/mwangaben/permission/storage"
	"github.com/mwangaben/permission/tenant"
)

// Manager wires the role sub-components together.
type Manager struct {
	repo        storage.Repository
	config      *config.Config
	tenant      *tenant.Manager
	Registrar   *Registrar
	Assigner    *Assigner
	PermManager *PermissionManager
}

func NewManager(repo storage.Repository, config *config.Config, tenant *tenant.Manager) *Manager {
	m := &Manager{
		repo:   repo,
		config: config,
		tenant: tenant,
	}
	m.wire()
	return m
}

func (m *Manager) wire() {
	m.Registrar = NewRegistrar(m.repo, m.config, m.tenant)
	m.Assigner = NewAssigner(m.repo)
	m.PermManager = NewPermissionManager(m.repo)
}

func (m *Manager) WithTenant(tenantID string) *Manager {
	newTenant := tenant.NewManager(m.tenant.IsEnabled()).WithTenant(tenantID)
	next := &Manager{
		repo:   m.repo,
		config: m.config,
		tenant: newTenant,
	}
	next.wire()
	return next
}
