package role

import (
	"github.com/google/uuid"
	"github.com/mwangaben/permission/models"
	"gorm.io/gorm"
)

// Registrar handles role registration
type Registrar struct {
	db *gorm.DB
}

// NewRegistrar creates a new role registrar
func NewRegistrar(db *gorm.DB) *Registrar {
	return &Registrar{db: db}
}

// Register creates a new role if it doesn't exist
func (r *Registrar) Register(role *models.Role) error {
	var existing models.Role

	query := r.db.Where("name = ?", role.Name)
	if role.TenantID != nil && *role.TenantID != "" {
		query = query.Where("tenant_id = ?", role.TenantID)
	} else {
		query = query.Where("tenant_id IS NULL")
	}

	if err := query.First(&existing).Error; err == nil {
		existing.DisplayName = role.DisplayName
		existing.Description = role.Description
		existing.IsDefault = role.IsDefault
		return r.db.Save(&existing).Error
	}

	if role.ID == "" {
		role.ID = uuid.New().String()
	}
	return r.db.Create(role).Error
}
