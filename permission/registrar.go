package permission

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mwangaben/permission/models"
	"gorm.io/gorm"
)

// Registrar handles permission registration
type Registrar struct {
	db *gorm.DB
}

// NewRegistrar creates a new permission registrar
func NewRegistrar(db *gorm.DB) *Registrar {
	return &Registrar{db: db}
}

// Register creates a new permission if it doesn't exist
func (r *Registrar) Register(permission *models.Permission) error {
	// Validate permission name format
	if !isValidPermissionName(permission.Name) {
		return fmt.Errorf("invalid permission name format: %s (expected {model}.{action})", permission.Name)
	}

	var existing models.Permission

	query := r.db.Where("name = ?", permission.Name)
	if permission.TenantID != nil && *permission.TenantID != "" {
		query = query.Where("tenant_id = ?", permission.TenantID)
	} else {
		query = query.Where("tenant_id IS NULL")
	}

	if err := query.First(&existing).Error; err == nil {
		// Permission already exists, update it
		existing.DisplayName = permission.DisplayName
		existing.Description = permission.Description
		existing.Model = permission.Model
		existing.Action = permission.Action
		return r.db.Save(&existing).Error
	}

	if permission.ID == "" {
		permission.ID = uuid.New().String()
	}
	return r.db.Create(permission).Error
}

// RegisterMany registers multiple permissions
func (r *Registrar) RegisterMany(permissions []models.Permission) error {
	for _, p := range permissions {
		if err := r.Register(&p); err != nil {
			return err
		}
	}
	return nil
}

// RegisterModelPermissions registers all standard CRUD permissions for a model
func (r *Registrar) RegisterModelPermissions(model, displayName string, tenantID *string) error {
	actions := []struct {
		action      string
		displayName string
		description string
	}{
		{"view", "View " + displayName, "Can view " + displayName},
		{"create", "Create " + displayName, "Can create " + displayName},
		{"update", "Update " + displayName, "Can update " + displayName},
		{"delete", "Delete " + displayName, "Can delete " + displayName},
	}

	for _, a := range actions {
		perm := &models.Permission{
			Name:        model + "." + a.action,
			Model:       model,
			Action:      a.action,
			DisplayName: a.displayName,
			Description: a.description,
			TenantID:    tenantID,
		}
		if err := r.Register(perm); err != nil {
			return err
		}
	}
	return nil
}

// isValidPermissionName checks if the permission name is in {model}.{action} format
func isValidPermissionName(name string) bool {
	// Simple validation - should contain a dot
	for i, c := range name {
		if c == '.' && i > 0 && i < len(name)-1 {
			return true
		}
	}
	return false
}

// Delete removes a permission
func (r *Registrar) Delete(name string, tenantID *string) error {
	query := r.db.Where("name = ?", name)
	if tenantID != nil && *tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	} else {
		query = query.Where("tenant_id IS NULL")
	}
	return query.Delete(&models.Permission{}).Error
}
