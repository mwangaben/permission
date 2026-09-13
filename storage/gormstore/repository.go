// Package gormstore implements storage.Repository on top of GORM.
//
// It consumes the models package directly and maps between models.*
// and storage.* types. No business logic lives here — only persistence.
package gormstore

import (
	"context"
	"errors"
	_ "fmt"
	"strings"
	"time"

	"github.com/mwangaben/permission/models"
	"github.com/mwangaben/permission/storage"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

// New returns a GORM-backed repository. db must not be nil.
func New(db interface{}) (storage.Repository, error) {
	gdb, ok := db.(*gorm.DB)
	if !ok {
		return nil, errors.New("gormstore: expected *gorm.DB")
	}
	return &Repository{db: gdb}, nil
}

func (r *Repository) Name() string { return storage.DriverGorm }

// ─── Migration ──────────────────────────────────────────────────────────

func (r *Repository) AutoMigrate(ctx context.Context) error {
	return r.db.WithContext(ctx).AutoMigrate(
		&models.Permission{},
		&models.Role{},
		&models.RoleHasPermission{},
		&models.ModelHasRole{},
		&models.ModelHasPermission{},
		&models.Tenant{},
		&models.TenantUser{},
	)
}

// ─── Permissions ────────────────────────────────────────────────────────

func (r *Repository) CreatePermission(ctx context.Context, p *storage.Permission) error {
	m := toGormPermission(p)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		if isUniqueViolation(err) {
			return storage.ErrPermissionExists
		}
		return err
	}
	p.ID = m.ID
	p.CreatedAt = m.CreatedAt
	p.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *Repository) FindPermissionByName(ctx context.Context, name, guard string, tenantID *string) (*storage.Permission, error) {
	var m models.Permission
	q := r.db.WithContext(ctx).
		Where("name = ? AND guard_name = ?", name, guard)
	q = applyTenantScope(q, "tenant_id", tenantID)
	if err := q.First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, storage.ErrPermissionNotFound
		}
		return nil, err
	}
	return fromGormPermission(&m), nil
}

func (r *Repository) FindPermissionByID(ctx context.Context, id uint) (*storage.Permission, error) {
	var m models.Permission
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, storage.ErrPermissionNotFound
		}
		return nil, err
	}
	return fromGormPermission(&m), nil
}

func (r *Repository) RolesWithPermission(ctx context.Context, permissionName, guardName string, tenantID *string) ([]*storage.Role, error) {
	var rows []models.Role

	q := r.db.WithContext(ctx).
		Table("roles").
		Joins("JOIN role_has_permissions ON role_has_permissions.role_id = roles.id").
		Joins("JOIN permissions ON permissions.id = role_has_permissions.permission_id").
		Where("permissions.name = ? AND permissions.guard_name = ?", permissionName, guardName).
		Where("roles.deleted_at IS NULL").
		Where("permissions.deleted_at IS NULL")

	q = applyTenantScope(q, "roles.tenant_id", tenantID)
	q = applyTenantScope(q, "permissions.tenant_id", tenantID)
	q = applyTenantScope(q, "role_has_permissions.tenant_id", tenantID)

	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return fromGormRoleSlice(rows), nil
}

func (r *Repository) SyncDirectPermissions(ctx context.Context, modelType string, modelID uint, permissionIDs []uint, tenantID *string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Remove all existing direct permissions for this model.
		if err := tx.Where("model_type = ? AND model_id = ?", modelType, modelID).
			Delete(&models.ModelHasPermission{}).Error; err != nil {
			return err
		}

		// Insert the new set.
		for _, permissionID := range permissionIDs {
			// Fetch the permission to determine its tenant for the pivot row.
			var perm models.Permission
			if err := tx.First(&perm, permissionID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return storage.ErrPermissionNotFound
				}
				return err
			}
			if err := validateTenantLink(perm.TenantID, tenantID); err != nil {
				return err
			}
			link := &models.ModelHasPermission{
				PermissionID: permissionID,
				ModelType:    modelType,
				ModelID:      modelID,
				TenantID:     perm.TenantID,
			}
			if err := tx.Create(link).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) SyncPermissionsForRole(ctx context.Context, roleID uint, permissionIDs []uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", roleID).
			Delete(&models.RoleHasPermission{}).Error; err != nil {
			return err
		}

		// Load the role once so we can validate each link.
		var role models.Role
		if err := tx.First(&role, roleID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return storage.ErrRoleNotFound
			}
			return err
		}

		for _, permissionID := range permissionIDs {
			var perm models.Permission
			if err := tx.First(&perm, permissionID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return storage.ErrPermissionNotFound
				}
				return err
			}
			if err := validateTenantLink(perm.TenantID, role.TenantID); err != nil {
				return err
			}
			link := &models.RoleHasPermission{
				PermissionID: permissionID,
				RoleID:       roleID,
				TenantID:     perm.TenantID,
			}
			if err := tx.Create(link).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) RevokeAllPermissionsForRole(ctx context.Context, roleID uint) error {
	return r.db.WithContext(ctx).
		Where("role_id = ?", roleID).
		Delete(&models.RoleHasPermission{}).Error
}

// ─── Roles ──────────────────────────────────────────────────────────────

func (r *Repository) CreateRole(ctx context.Context, role *storage.Role) error {
	m := toGormRole(role)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		if isUniqueViolation(err) {
			return storage.ErrRoleExists
		}
		return err
	}
	role.ID = m.ID
	role.CreatedAt = m.CreatedAt
	role.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *Repository) FindRoleByName(ctx context.Context, name, guard string, tenantID *string) (*storage.Role, error) {
	var m models.Role
	q := r.db.WithContext(ctx).
		Where("name = ? AND guard_name = ?", name, guard)
	q = applyTenantScope(q, "tenant_id", tenantID)
	if err := q.First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, storage.ErrRoleNotFound
		}
		return nil, err
	}
	return fromGormRole(&m), nil
}

func (r *Repository) FindRoleByID(ctx context.Context, id uint) (*storage.Role, error) {
	var m models.Role
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, storage.ErrRoleNotFound
		}
		return nil, err
	}
	return fromGormRole(&m), nil
}

// ─── Role ↔ Permission ──────────────────────────────────────────────────

func (r *Repository) AssignPermissionToRole(ctx context.Context, permissionID, roleID uint) error {
	perm, err := r.FindPermissionByID(ctx, permissionID)
	if err != nil {
		return err
	}
	role, err := r.FindRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if err := validateTenantLink(perm.TenantID, role.TenantID); err != nil {
		return err
	}

	link := &models.RoleHasPermission{
		PermissionID: permissionID,
		RoleID:       roleID,
		TenantID:     perm.TenantID, // permission's tenant defines the link
	}
	if err := r.db.WithContext(ctx).
		Where("permission_id = ? AND role_id = ?", permissionID, roleID).
		FirstOrCreate(link).Error; err != nil {
		return err
	}
	return nil
}

func (r *Repository) RemovePermissionFromRole(ctx context.Context, permissionID, roleID uint) error {
	return r.db.WithContext(ctx).
		Where("permission_id = ? AND role_id = ?", permissionID, roleID).
		Delete(&models.RoleHasPermission{}).Error
}

func (r *Repository) GetPermissionsForRole(ctx context.Context, roleID uint) ([]*storage.Permission, error) {
	var rows []models.Permission
	err := r.db.WithContext(ctx).
		Table("permissions").
		Joins("JOIN role_has_permissions ON role_has_permissions.permission_id = permissions.id").
		Where("role_has_permissions.role_id = ?", roleID).
		Where("permissions.deleted_at IS NULL").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return fromGormPermissionSlice(rows), nil
}

// ─── Model ↔ Role ───────────────────────────────────────────────────────

func (r *Repository) AssignRoleToModel(ctx context.Context, roleID uint, modelType string, modelID uint, tenantID *string) error {
	role, err := r.FindRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if err := validateTenantLink(role.TenantID, tenantID); err != nil {
		return err
	}

	link := &models.ModelHasRole{
		RoleID:    roleID,
		ModelType: modelType,
		ModelID:   modelID,
		TenantID:  role.TenantID, // role's tenant defines the link
	}
	return r.db.WithContext(ctx).
		Where("role_id = ? AND model_type = ? AND model_id = ?", roleID, modelType, modelID).
		FirstOrCreate(link).Error
}

func (r *Repository) RemoveRoleFromModel(ctx context.Context, roleID uint, modelType string, modelID uint) error {
	return r.db.WithContext(ctx).
		Where("role_id = ? AND model_type = ? AND model_id = ?", roleID, modelType, modelID).
		Delete(&models.ModelHasRole{}).Error
}

func (r *Repository) GetRolesForModel(ctx context.Context, modelType string, modelID uint, tenantID *string) ([]*storage.Role, error) {
	var rows []models.Role
	q := r.db.WithContext(ctx).
		Table("roles").
		Joins("JOIN model_has_roles ON model_has_roles.role_id = roles.id").
		Where("model_has_roles.model_type = ? AND model_has_roles.model_id = ?", modelType, modelID).
		Where("roles.deleted_at IS NULL")

	q = applyTenantScope(q, "roles.tenant_id", tenantID)
	q = applyTenantScope(q, "model_has_roles.tenant_id", tenantID)

	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return fromGormRoleSlice(rows), nil
}

func (r *Repository) SyncRolesForModel(ctx context.Context, modelType string, modelID uint, roleIDs []uint, tenantID *string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("model_type = ? AND model_id = ?", modelType, modelID).
			Delete(&models.ModelHasRole{}).Error; err != nil {
			return err
		}
		for _, roleID := range roleIDs {
			role, err := r.FindRoleByID(ctx, roleID)
			if err != nil {
				return err
			}
			if err := validateTenantLink(role.TenantID, tenantID); err != nil {
				return err
			}
			link := &models.ModelHasRole{
				RoleID:    roleID,
				ModelType: modelType,
				ModelID:   modelID,
				TenantID:  role.TenantID,
			}
			if err := tx.Create(link).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ─── Model ↔ Permission (direct) ────────────────────────────────────────

func (r *Repository) AssignPermissionToModel(ctx context.Context, permissionID uint, modelType string, modelID uint, tenantID *string) error {
	perm, err := r.FindPermissionByID(ctx, permissionID)
	if err != nil {
		return err
	}
	if err := validateTenantLink(perm.TenantID, tenantID); err != nil {
		return err
	}

	link := &models.ModelHasPermission{
		PermissionID: permissionID,
		ModelType:    modelType,
		ModelID:      modelID,
		TenantID:     perm.TenantID,
	}
	return r.db.WithContext(ctx).
		Where("permission_id = ? AND model_type = ? AND model_id = ?", permissionID, modelType, modelID).
		FirstOrCreate(link).Error
}

func (r *Repository) RemovePermissionFromModel(ctx context.Context, permissionID uint, modelType string, modelID uint) error {
	return r.db.WithContext(ctx).
		Where("permission_id = ? AND model_type = ? AND model_id = ?", permissionID, modelType, modelID).
		Delete(&models.ModelHasPermission{}).Error
}

func (r *Repository) GetDirectPermissionsForModel(ctx context.Context, modelType string, modelID uint, guardName string, tenantID *string) ([]*storage.Permission, error) {
	var rows []models.Permission
	q := r.db.WithContext(ctx).
		Table("permissions").
		Joins("JOIN model_has_permissions ON model_has_permissions.permission_id = permissions.id").
		Where("permissions.guard_name = ?", guardName).
		Where("model_has_permissions.model_type = ? AND model_has_permissions.model_id = ?", modelType, modelID).
		Where("permissions.deleted_at IS NULL")

	q = applyTenantScope(q, "permissions.tenant_id", tenantID)
	q = applyTenantScope(q, "model_has_permissions.tenant_id", tenantID)

	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return fromGormPermissionSlice(rows), nil
}

// ─── Cross-table permission checks ──────────────────────────────────────

func (r *Repository) DirectPermissionCount(ctx context.Context, modelType string, modelID uint, guardName string, names []string, tenantID *string) (int, error) {
	q := r.db.WithContext(ctx).
		Table("permissions").
		Joins("JOIN model_has_permissions ON model_has_permissions.permission_id = permissions.id").
		Where("permissions.guard_name = ?", guardName).
		Where("model_has_permissions.model_type = ? AND model_has_permissions.model_id = ?", modelType, modelID).
		Where("permissions.deleted_at IS NULL")

	if len(names) > 0 {
		q = q.Where("permissions.name IN ?", names)
	}
	q = applyTenantScope(q, "permissions.tenant_id", tenantID)
	q = applyTenantScope(q, "model_has_permissions.tenant_id", tenantID)

	var count int64
	if err := q.Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (r *Repository) RolePermissionCount(ctx context.Context, modelType string, modelID uint, guardName string, names []string, tenantID *string) (int, error) {
	q := r.db.WithContext(ctx).
		Table("permissions").
		Joins("JOIN role_has_permissions ON role_has_permissions.permission_id = permissions.id").
		Joins("JOIN roles ON roles.id = role_has_permissions.role_id").
		Joins("JOIN model_has_roles ON model_has_roles.role_id = roles.id").
		Where("permissions.guard_name = ?", guardName).
		Where("model_has_roles.model_type = ? AND model_has_roles.model_id = ?", modelType, modelID).
		Where("permissions.deleted_at IS NULL").
		Where("roles.deleted_at IS NULL")

	if len(names) > 0 {
		q = q.Where("permissions.name IN ?", names)
	}
	q = applyTenantScope(q, "permissions.tenant_id", tenantID)
	q = applyTenantScope(q, "roles.tenant_id", tenantID)
	q = applyTenantScope(q, "role_has_permissions.tenant_id", tenantID)
	q = applyTenantScope(q, "model_has_roles.tenant_id", tenantID)

	var count int64
	if err := q.Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (r *Repository) GetRoleDerivedPermissionsForModel(ctx context.Context, modelType string, modelID uint, guardName string, tenantID *string) ([]*storage.Permission, error) {
	var rows []models.Permission
	q := r.db.WithContext(ctx).
		Table("permissions").
		Joins("JOIN role_has_permissions ON role_has_permissions.permission_id = permissions.id").
		Joins("JOIN roles ON roles.id = role_has_permissions.role_id").
		Joins("JOIN model_has_roles ON model_has_roles.role_id = roles.id").
		Where("permissions.guard_name = ?", guardName).
		Where("model_has_roles.model_type = ? AND model_has_roles.model_id = ?", modelType, modelID).
		Where("permissions.deleted_at IS NULL").
		Where("roles.deleted_at IS NULL")

	q = applyTenantScope(q, "permissions.tenant_id", tenantID)
	q = applyTenantScope(q, "roles.tenant_id", tenantID)
	q = applyTenantScope(q, "role_has_permissions.tenant_id", tenantID)
	q = applyTenantScope(q, "model_has_roles.tenant_id", tenantID)

	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return fromGormPermissionSlice(rows), nil
}

// ─── Tenants ────────────────────────────────────────────────────────────

func (r *Repository) CreateTenant(ctx context.Context, t *storage.Tenant) error {
	m := toGormTenant(t)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		if isUniqueViolation(err) {
			return storage.ErrTenantExists
		}
		return err
	}
	t.CreatedAt = m.CreatedAt
	t.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *Repository) FindTenantByID(ctx context.Context, id string) (*storage.Tenant, error) {
	var m models.Tenant
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, storage.ErrTenantNotFound
		}
		return nil, err
	}
	return fromGormTenant(&m), nil
}

func (r *Repository) FindTenantBySlug(ctx context.Context, slug string) (*storage.Tenant, error) {
	var m models.Tenant
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, storage.ErrTenantNotFound
		}
		return nil, err
	}
	return fromGormTenant(&m), nil
}

func (r *Repository) AssignUserToTenant(ctx context.Context, userID, tenantID string) error {
	link := &models.TenantUser{UserID: userID, TenantID: tenantID}
	return r.db.WithContext(ctx).
		Where("user_id = ? AND tenant_id = ?", userID, tenantID).
		FirstOrCreate(link).Error
}

func (r *Repository) RemoveUserFromTenant(ctx context.Context, userID, tenantID string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND tenant_id = ?", userID, tenantID).
		Delete(&models.TenantUser{}).Error
}

func (r *Repository) GetTenantsForUser(ctx context.Context, userID string) ([]*storage.Tenant, error) {
	var rows []models.Tenant
	err := r.db.WithContext(ctx).
		Table("tenants").
		Joins("JOIN tenant_user ON tenant_user.tenant_id = tenants.id").
		Where("tenant_user.user_id = ?", userID).
		Where("tenants.deleted_at IS NULL").
		Where("tenant_user.deleted_at IS NULL").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return fromGormTenantSlice(rows), nil
}

// ─── Internal helpers ───────────────────────────────────────────────────

// applyTenantScope adds the standard tenant filter to a query:
//   - nil or empty tenantID → tenant_id IS NULL  (global only)
//   - non-empty tenantID    → tenant_id = :id OR tenant_id IS NULL
//
// column is the fully-qualified column name (e.g. "permissions.tenant_id").
func applyTenantScope(q *gorm.DB, column string, tenantID *string) *gorm.DB {
	if tenantID == nil || *tenantID == "" {
		return q.Where(column + " IS NULL")
	}
	return q.Where("("+column+" = ? OR "+column+" IS NULL)", *tenantID)
}

// validateTenantLink enforces the strict cross-tenant rule:
//   - both global        → OK
//   - one global         → OK (globals are shareable)
//   - same tenant        → OK
//   - different tenants  → ErrCrossTenantAssignment
func validateTenantLink(a, b *string) error {
	aScoped := a != nil && *a != ""
	bScoped := b != nil && *b != ""
	if !aScoped || !bScoped {
		return nil
	}
	if *a != *b {
		return storage.ErrCrossTenantAssignment
	}
	return nil
}

// isUniqueViolation checks if err is a MySQL/Postgres unique-constraint error.
// Covers the common driver messages; expand if you add more dialects.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "unique violation")
}

// ─── Mappers: models.* → storage.* ──────────────────────────────────────

func toGormPermission(p *storage.Permission) *models.Permission {
	return &models.Permission{
		ID:        p.ID,
		Name:      p.Name,
		GuardName: p.GuardName,
		TenantID:  p.TenantID,
	}
}

func fromGormPermission(m *models.Permission) *storage.Permission {
	return &storage.Permission{
		ID:        m.ID,
		Name:      m.Name,
		GuardName: m.GuardName,
		TenantID:  m.TenantID,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: toTimePtr(m.DeletedAt),
	}
}

func fromGormPermissionSlice(ms []models.Permission) []*storage.Permission {
	out := make([]*storage.Permission, 0, len(ms))
	for i := range ms {
		out = append(out, fromGormPermission(&ms[i]))
	}
	return out
}

func toGormRole(r *storage.Role) *models.Role {
	return &models.Role{
		ID:        r.ID,
		Name:      r.Name,
		GuardName: r.GuardName,
		TenantID:  r.TenantID,
	}
}

func fromGormRole(m *models.Role) *storage.Role {
	return &storage.Role{
		ID:        m.ID,
		Name:      m.Name,
		GuardName: m.GuardName,
		TenantID:  m.TenantID,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: toTimePtr(m.DeletedAt),
	}
}

func fromGormRoleSlice(ms []models.Role) []*storage.Role {
	out := make([]*storage.Role, 0, len(ms))
	for i := range ms {
		out = append(out, fromGormRole(&ms[i]))
	}
	return out
}

func toGormTenant(t *storage.Tenant) *models.Tenant {
	return &models.Tenant{
		ID:       t.ID,
		Name:     t.Name,
		Slug:     t.Slug,
		Domain:   t.Domain,
		Logo:     t.Logo,
		Active:   t.Active,
		OwnerID:  t.OwnerID,
		Config:   t.Config,
		Settings: t.Settings,
	}
}

func fromGormTenant(m *models.Tenant) *storage.Tenant {
	return &storage.Tenant{
		ID:        m.ID,
		Name:      m.Name,
		Slug:      m.Slug,
		Domain:    m.Domain,
		Logo:      m.Logo,
		Active:    m.Active,
		OwnerID:   m.OwnerID,
		Config:    m.Config,
		Settings:  m.Settings,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: toTimePtr(m.DeletedAt),
	}
}

func fromGormTenantSlice(ms []models.Tenant) []*storage.Tenant {
	out := make([]*storage.Tenant, 0, len(ms))
	for i := range ms {
		out = append(out, fromGormTenant(&ms[i]))
	}
	return out
}

func (r *Repository) RolePermissionExists(ctx context.Context, roleID uint, permissionName, guardName string, tenantID *string) (bool, error) {
	q := r.db.WithContext(ctx).
		Table("role_has_permissions").
		Joins("JOIN permissions ON permissions.id = role_has_permissions.permission_id").
		Where("role_has_permissions.role_id = ?", roleID).
		Where("permissions.name = ? AND permissions.guard_name = ?", permissionName, guardName).
		Where("permissions.deleted_at IS NULL")

	q = applyTenantScope(q, "permissions.tenant_id", tenantID)
	q = applyTenantScope(q, "role_has_permissions.tenant_id", tenantID)

	var count int64
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// toTimePtr converts gorm.DeletedAt into *time.Time (nil when not set).
func toTimePtr(d gorm.DeletedAt) *time.Time {
	if !d.Valid {
		return nil
	}
	t := d.Time
	return &t
}

// Assert at compile time that *Repository satisfies storage.Repository.
var _ storage.Repository = (*Repository)(nil)
