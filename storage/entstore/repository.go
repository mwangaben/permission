package entstore

import (
	"context"
	"errors"

	"github.com/mwangaben/permission/storage"
	"github.com/mwangaben/permission/storage/entstore/ent"
	"github.com/mwangaben/permission/storage/entstore/ent/modelhaspermission"
	"github.com/mwangaben/permission/storage/entstore/ent/modelhasrole"
	"github.com/mwangaben/permission/storage/entstore/ent/permission"
	"github.com/mwangaben/permission/storage/entstore/ent/predicate"
	"github.com/mwangaben/permission/storage/entstore/ent/role"
	"github.com/mwangaben/permission/storage/entstore/ent/rolehaspermission"
	"github.com/mwangaben/permission/storage/entstore/ent/tenant"
	"github.com/mwangaben/permission/storage/entstore/ent/tenantuser"
)

type Repository struct {
	client *ent.Client
}

func New(db interface{}) (storage.Repository, error) {
	switch v := db.(type) {
	case *ent.Client:
		return &Repository{client: v}, nil
	case interface{ Client() *ent.Client }:
		return &Repository{client: v.Client()}, nil
	default:
		return nil, errors.New("entstore: expected *ent.Client or a value exposing Client() *ent.Client")
	}
}

func (r *Repository) Name() string { return storage.DriverEnt }

func (r *Repository) AutoMigrate(ctx context.Context) error {
	return r.client.Schema.Create(ctx)
}

// ─── Permissions ────────────────────────────────────────────────────────

func (r *Repository) CreatePermission(ctx context.Context, p *storage.Permission) error {
	b := r.client.Permission.Create().
		SetName(p.Name).
		SetGuardName(p.GuardName)

	if p.TenantID != nil {
		b = b.SetTenantID(*p.TenantID)
	}
	if p.ID != 0 {
		b = b.SetID(p.ID)
	}

	m, err := b.Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
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
	q := r.client.Permission.Query().
		Where(
			permission.Name(name),
			permission.GuardName(guard),
			permission.DeletedAtIsNil(),
		)
	q = applyPermissionTenantScope(q, tenantID)

	m, err := q.Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, storage.ErrPermissionNotFound
		}
		return nil, err
	}
	return fromEntPermission(m), nil
}

func (r *Repository) FindPermissionByID(ctx context.Context, id uint) (*storage.Permission, error) {
	m, err := r.client.Permission.Query().
		Where(permission.ID(id), permission.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, storage.ErrPermissionNotFound
		}
		return nil, err
	}
	return fromEntPermission(m), nil
}

// ─── Roles ──────────────────────────────────────────────────────────────

func (r *Repository) CreateRole(ctx context.Context, role *storage.Role) error {
	b := r.client.Role.Create().
		SetName(role.Name).
		SetGuardName(role.GuardName)

	if role.TenantID != nil {
		b = b.SetTenantID(*role.TenantID)
	}
	if role.ID != 0 {
		b = b.SetID(role.ID)
	}

	m, err := b.Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
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
	q := r.client.Role.Query().
		Where(
			role.Name(name),
			role.GuardName(guard),
			role.DeletedAtIsNil(),
		)
	q = applyRoleTenantScope(q, tenantID)

	m, err := q.Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, storage.ErrRoleNotFound
		}
		return nil, err
	}
	return fromEntRole(m), nil
}

func (r *Repository) FindRoleByID(ctx context.Context, id uint) (*storage.Role, error) {
	m, err := r.client.Role.Query().
		Where(role.ID(id), role.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, storage.ErrRoleNotFound
		}
		return nil, err
	}
	return fromEntRole(m), nil
}

// ─── Role ↔ Permission ──────────────────────────────────────────────────

func (r *Repository) AssignPermissionToRole(ctx context.Context, permissionID, roleID uint) error {
	perm, err := r.FindPermissionByID(ctx, permissionID)
	if err != nil {
		return err
	}
	rl, err := r.FindRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if err := validateTenantLink(perm.TenantID, rl.TenantID); err != nil {
		return err
	}

	// Idempotent insert: check first, then create.
	exists, err := r.client.RoleHasPermission.Query().
		Where(
			rolehaspermission.PermissionID(permissionID),
			rolehaspermission.RoleID(roleID),
		).
		Exist(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	b := r.client.RoleHasPermission.Create().
		SetPermissionID(permissionID).
		SetRoleID(roleID)
	if perm.TenantID != nil {
		b = b.SetTenantID(*perm.TenantID)
	}
	_, err = b.Save(ctx)
	return err
}

func (r *Repository) RemovePermissionFromRole(ctx context.Context, permissionID, roleID uint) error {
	_, err := r.client.RoleHasPermission.Delete().
		Where(
			rolehaspermission.PermissionID(permissionID),
			rolehaspermission.RoleID(roleID),
		).
		Exec(ctx)
	return err
}

func (r *Repository) GetPermissionsForRole(ctx context.Context, roleID uint) ([]*storage.Permission, error) {
	// Select permissions where there exists a role_has_permissions linking
	// them to roleID. Expressed as a subquery on the pivot table.
	pivotPermissionIDs, err := r.client.RoleHasPermission.Query().
		Where(rolehaspermission.RoleID(roleID)).
		Select(rolehaspermission.FieldPermissionID).
		Ints(ctx)
	if err != nil {
		return nil, err
	}
	if len(pivotPermissionIDs) == 0 {
		return nil, nil
	}

	rows, err := r.client.Permission.Query().
		Where(
			permission.IDIn(idsToUints(pivotPermissionIDs)...),
			permission.DeletedAtIsNil(),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return fromEntPermissionSlice(rows), nil
}

// ─── Model ↔ Role ───────────────────────────────────────────────────────

func (r *Repository) AssignRoleToModel(ctx context.Context, roleID uint, modelType string, modelID uint, tenantID *string) error {
	rl, err := r.FindRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if err := validateTenantLink(rl.TenantID, tenantID); err != nil {
		return err
	}

	exists, err := r.client.ModelHasRole.Query().
		Where(
			modelhasrole.RoleID(roleID),
			modelhasrole.ModelType(modelType),
			modelhasrole.ModelID(modelID),
		).
		Exist(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	b := r.client.ModelHasRole.Create().
		SetRoleID(roleID).
		SetModelType(modelType).
		SetModelID(modelID)
	if rl.TenantID != nil {
		b = b.SetTenantID(*rl.TenantID)
	}
	_, err = b.Save(ctx)
	return err
}

func (r *Repository) RemoveRoleFromModel(ctx context.Context, roleID uint, modelType string, modelID uint) error {
	_, err := r.client.ModelHasRole.Delete().
		Where(
			modelhasrole.RoleID(roleID),
			modelhasrole.ModelType(modelType),
			modelhasrole.ModelID(modelID),
		).
		Exec(ctx)
	return err
}

func (r *Repository) GetRolesForModel(ctx context.Context, modelType string, modelID uint, tenantID *string) ([]*storage.Role, error) {
	pivotRoleIDs, err := r.client.ModelHasRole.Query().
		Where(
			modelhasrole.ModelType(modelType),
			modelhasrole.ModelID(modelID),
		).
		Where(applyModelHasRoleTenantPredicate(tenantID)).
		Select(modelhasrole.FieldRoleID).
		Ints(ctx)
	if err != nil {
		return nil, err
	}
	if len(pivotRoleIDs) == 0 {
		return nil, nil
	}

	q := r.client.Role.Query().
		Where(
			role.IDIn(idsToUints(pivotRoleIDs)...),
			role.DeletedAtIsNil(),
		)
	q = applyRoleTenantScope(q, tenantID)

	rows, err := q.All(ctx)
	if err != nil {
		return nil, err
	}
	return fromEntRoleSlice(rows), nil
}

func (r *Repository) SyncRolesForModel(ctx context.Context, modelType string, modelID uint, roleIDs []uint, tenantID *string) error {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}

	if _, err := tx.ModelHasRole.Delete().
		Where(
			modelhasrole.ModelType(modelType),
			modelhasrole.ModelID(modelID),
		).
		Exec(ctx); err != nil {
		return rollback(tx, err)
	}

	for _, roleID := range roleIDs {
		rl, err := r.FindRoleByID(ctx, roleID)
		if err != nil {
			return rollback(tx, err)
		}
		if err := validateTenantLink(rl.TenantID, tenantID); err != nil {
			return rollback(tx, err)
		}
		b := tx.ModelHasRole.Create().
			SetRoleID(roleID).
			SetModelType(modelType).
			SetModelID(modelID)
		if rl.TenantID != nil {
			b = b.SetTenantID(*rl.TenantID)
		}
		if _, err := b.Save(ctx); err != nil {
			return rollback(tx, err)
		}
	}
	return tx.Commit()
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

	exists, err := r.client.ModelHasPermission.Query().
		Where(
			modelhaspermission.PermissionID(permissionID),
			modelhaspermission.ModelType(modelType),
			modelhaspermission.ModelID(modelID),
		).
		Exist(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	b := r.client.ModelHasPermission.Create().
		SetPermissionID(permissionID).
		SetModelType(modelType).
		SetModelID(modelID)
	if perm.TenantID != nil {
		b = b.SetTenantID(*perm.TenantID)
	}
	_, err = b.Save(ctx)
	return err
}

func (r *Repository) RemovePermissionFromModel(ctx context.Context, permissionID uint, modelType string, modelID uint) error {
	_, err := r.client.ModelHasPermission.Delete().
		Where(
			modelhaspermission.PermissionID(permissionID),
			modelhaspermission.ModelType(modelType),
			modelhaspermission.ModelID(modelID),
		).
		Exec(ctx)
	return err
}

func (r *Repository) GetDirectPermissionsForModel(ctx context.Context, modelType string, modelID uint, guardName string, tenantID *string) ([]*storage.Permission, error) {
	pivotPermissionIDs, err := r.client.ModelHasPermission.Query().
		Where(
			modelhaspermission.ModelType(modelType),
			modelhaspermission.ModelID(modelID),
		).
		Where(applyModelHasPermissionTenantPredicate(tenantID)).
		Select(modelhaspermission.FieldPermissionID).
		Ints(ctx)
	if err != nil {
		return nil, err
	}
	if len(pivotPermissionIDs) == 0 {
		return nil, nil
	}

	q := r.client.Permission.Query().
		Where(
			permission.IDIn(idsToUints(pivotPermissionIDs)...),
			permission.GuardName(guardName),
			permission.DeletedAtIsNil(),
		)
	q = applyPermissionTenantScope(q, tenantID)

	rows, err := q.All(ctx)
	if err != nil {
		return nil, err
	}
	return fromEntPermissionSlice(rows), nil
}

// ─── Cross-table permission checks ──────────────────────────────────────

func (r *Repository) DirectPermissionCount(ctx context.Context, modelType string, modelID uint, guardName string, names []string, tenantID *string) (int, error) {
	pivotPermissionIDs, err := r.client.ModelHasPermission.Query().
		Where(
			modelhaspermission.ModelType(modelType),
			modelhaspermission.ModelID(modelID),
		).
		Where(applyModelHasPermissionTenantPredicate(tenantID)).
		Select(modelhaspermission.FieldPermissionID).
		Ints(ctx)
	if err != nil {
		return 0, err
	}
	if len(pivotPermissionIDs) == 0 {
		return 0, nil
	}

	q := r.client.Permission.Query().
		Where(
			permission.IDIn(idsToUints(pivotPermissionIDs)...),
			permission.GuardName(guardName),
			permission.DeletedAtIsNil(),
		)
	if len(names) > 0 {
		q = q.Where(permission.NameIn(names...))
	}
	q = applyPermissionTenantScope(q, tenantID)

	return q.Count(ctx)
}

func (r *Repository) RolePermissionCount(ctx context.Context, modelType string, modelID uint, guardName string, names []string, tenantID *string) (int, error) {
	// Step 1: role IDs assigned to the model.
	roleIDs, err := r.client.ModelHasRole.Query().
		Where(
			modelhasrole.ModelType(modelType),
			modelhasrole.ModelID(modelID),
		).
		Where(applyModelHasRoleTenantPredicate(tenantID)).
		Select(modelhasrole.FieldRoleID).
		Ints(ctx)
	if err != nil {
		return 0, err
	}
	if len(roleIDs) == 0 {
		return 0, nil
	}

	// Step 2: permission IDs granted to those roles.
	permIDs, err := r.client.RoleHasPermission.Query().
		Where(rolehaspermission.RoleIDIn(idsToUints(roleIDs)...)).
		Where(applyRoleHasPermissionTenantPredicate(tenantID)).
		Select(rolehaspermission.FieldPermissionID).
		Ints(ctx)
	if err != nil {
		return 0, err
	}
	if len(permIDs) == 0 {
		return 0, nil
	}

	// Step 3: count matching permissions.
	q := r.client.Permission.Query().
		Where(
			permission.IDIn(idsToUints(permIDs)...),
			permission.GuardName(guardName),
			permission.DeletedAtIsNil(),
		)
	if len(names) > 0 {
		q = q.Where(permission.NameIn(names...))
	}
	q = applyPermissionTenantScope(q, tenantID)

	return q.Count(ctx)
}

func (r *Repository) GetRoleDerivedPermissionsForModel(ctx context.Context, modelType string, modelID uint, guardName string, tenantID *string) ([]*storage.Permission, error) {
	roleIDs, err := r.client.ModelHasRole.Query().
		Where(
			modelhasrole.ModelType(modelType),
			modelhasrole.ModelID(modelID),
		).
		Where(applyModelHasRoleTenantPredicate(tenantID)).
		Select(modelhasrole.FieldRoleID).
		Ints(ctx)
	if err != nil {
		return nil, err
	}
	if len(roleIDs) == 0 {
		return nil, nil
	}

	permIDs, err := r.client.RoleHasPermission.Query().
		Where(rolehaspermission.RoleIDIn(idsToUints(roleIDs)...)).
		Where(applyRoleHasPermissionTenantPredicate(tenantID)).
		Select(rolehaspermission.FieldPermissionID).
		Ints(ctx)
	if err != nil {
		return nil, err
	}
	if len(permIDs) == 0 {
		return nil, nil
	}

	q := r.client.Permission.Query().
		Where(
			permission.IDIn(idsToUints(permIDs)...),
			permission.GuardName(guardName),
			permission.DeletedAtIsNil(),
		)
	q = applyPermissionTenantScope(q, tenantID)

	rows, err := q.All(ctx)
	if err != nil {
		return nil, err
	}
	return fromEntPermissionSlice(rows), nil
}

// ─── Tenants ────────────────────────────────────────────────────────────

func (r *Repository) CreateTenant(ctx context.Context, t *storage.Tenant) error {
	b := r.client.Tenant.Create().
		SetID(t.ID).
		SetName(t.Name).
		SetSlug(t.Slug).
		SetDomain(t.Domain).
		SetLogo(t.Logo).
		SetActive(t.Active).
		SetOwnerID(t.OwnerID).
		SetConfig(t.Config).
		SetSettings(t.Settings)

	m, err := b.Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return storage.ErrTenantExists
		}
		return err
	}
	t.CreatedAt = m.CreatedAt
	t.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *Repository) FindTenantByID(ctx context.Context, id string) (*storage.Tenant, error) {
	m, err := r.client.Tenant.Query().
		Where(tenant.ID(id), tenant.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, storage.ErrTenantNotFound
		}
		return nil, err
	}
	return fromEntTenant(m), nil
}

func (r *Repository) FindTenantBySlug(ctx context.Context, slug string) (*storage.Tenant, error) {
	m, err := r.client.Tenant.Query().
		Where(tenant.Slug(slug), tenant.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, storage.ErrTenantNotFound
		}
		return nil, err
	}
	return fromEntTenant(m), nil
}

func (r *Repository) AssignUserToTenant(ctx context.Context, userID, tenantID string) error {
	exists, err := r.client.TenantUser.Query().
		Where(
			tenantuser.UserID(userID),
			tenantuser.TenantID(tenantID),
			tenantuser.DeletedAtIsNil(),
		).
		Exist(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = r.client.TenantUser.Create().
		SetUserID(userID).
		SetTenantID(tenantID).
		Save(ctx)
	return err
}

func (r *Repository) RemoveUserFromTenant(ctx context.Context, userID, tenantID string) error {
	_, err := r.client.TenantUser.Delete().
		Where(
			tenantuser.UserID(userID),
			tenantuser.TenantID(tenantID),
		).
		Exec(ctx)
	return err
}

func (r *Repository) GetTenantsForUser(ctx context.Context, userID string) ([]*storage.Tenant, error) {
	tenantIDs, err := r.client.TenantUser.Query().
		Where(
			tenantuser.UserID(userID),
			tenantuser.DeletedAtIsNil(),
		).
		Select(tenantuser.FieldTenantID).
		Strings(ctx)
	if err != nil {
		return nil, err
	}
	if len(tenantIDs) == 0 {
		return nil, nil
	}

	rows, err := r.client.Tenant.Query().
		Where(
			tenant.IDIn(tenantIDs...),
			tenant.DeletedAtIsNil(),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return fromEntTenantSlice(rows), nil
}

// ─── Internal helpers ───────────────────────────────────────────────────

func applyPermissionTenantScope(q *ent.PermissionQuery, tenantID *string) *ent.PermissionQuery {
	if tenantID == nil || *tenantID == "" {
		return q.Where(permission.TenantIDIsNil())
	}
	return q.Where(
		permission.Or(
			permission.TenantID(*tenantID),
			permission.TenantIDIsNil(),
		),
	)
}

func applyRoleTenantScope(q *ent.RoleQuery, tenantID *string) *ent.RoleQuery {
	if tenantID == nil || *tenantID == "" {
		return q.Where(role.TenantIDIsNil())
	}
	return q.Where(
		role.Or(
			role.TenantID(*tenantID),
			role.TenantIDIsNil(),
		),
	)
}

// validateTenantLink enforces the strict cross-tenant rule. Same logic as
// gormstore; duplicated here to avoid a shared package just for this.
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

func rollback(tx *ent.Tx, err error) error {
	if rbErr := tx.Rollback(); rbErr != nil {
		return rbErr
	}
	return err
}

// ─── Mappers: ent.* → storage.* ─────────────────────────────────────────

func fromEntPermission(m *ent.Permission) *storage.Permission {
	return &storage.Permission{
		ID:        m.ID,
		Name:      m.Name,
		GuardName: m.GuardName,
		TenantID:  m.TenantID,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}
}

func fromEntPermissionSlice(ms []*ent.Permission) []*storage.Permission {
	out := make([]*storage.Permission, 0, len(ms))
	for _, m := range ms {
		out = append(out, fromEntPermission(m))
	}
	return out
}

func fromEntRole(m *ent.Role) *storage.Role {
	return &storage.Role{
		ID:        m.ID,
		Name:      m.Name,
		GuardName: m.GuardName,
		TenantID:  m.TenantID,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}
}

func fromEntRoleSlice(ms []*ent.Role) []*storage.Role {
	out := make([]*storage.Role, 0, len(ms))
	for _, m := range ms {
		out = append(out, fromEntRole(m))
	}
	return out
}

func fromEntTenant(m *ent.Tenant) *storage.Tenant {
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
		DeletedAt: m.DeletedAt,
	}
}

func fromEntTenantSlice(ms []*ent.Tenant) []*storage.Tenant {
	out := make([]*storage.Tenant, 0, len(ms))
	for _, m := range ms {
		out = append(out, fromEntTenant(m))
	}
	return out
}

func applyRoleHasPermissionTenantPredicate(tenantID *string) predicate.RoleHasPermission {
	if tenantID == nil || *tenantID == "" {
		return rolehaspermission.TenantIDIsNil()
	}
	return rolehaspermission.Or(
		rolehaspermission.TenantID(*tenantID),
		rolehaspermission.TenantIDIsNil(),
	)
}

func applyModelHasRoleTenantPredicate(tenantID *string) predicate.ModelHasRole {
	if tenantID == nil || *tenantID == "" {
		return modelhasrole.TenantIDIsNil()
	}
	return modelhasrole.Or(
		modelhasrole.TenantID(*tenantID),
		modelhasrole.TenantIDIsNil(),
	)
}

func applyModelHasPermissionTenantPredicate(tenantID *string) predicate.ModelHasPermission {
	if tenantID == nil || *tenantID == "" {
		return modelhaspermission.TenantIDIsNil()
	}
	return modelhaspermission.Or(
		modelhaspermission.TenantID(*tenantID),
		modelhaspermission.TenantIDIsNil(),
	)
}

func (r *Repository) RolesWithPermission(ctx context.Context, permissionName, guardName string, tenantID *string) ([]*storage.Role, error) {
	// Step 1: find the permission IDs matching (name, guard).
	permIDs, err := r.client.Permission.Query().
		Where(
			permission.Name(permissionName),
			permission.GuardName(guardName),
			permission.DeletedAtIsNil(),
		).
		Where(applyPermissionTenantScopePredicate(tenantID)).
		IDs(ctx)
	if err != nil {
		return nil, err
	}
	if len(permIDs) == 0 {
		return nil, nil
	}

	// Step 2: find the role IDs linked to those permissions.
	roleIDs, err := r.client.RoleHasPermission.Query().
		Where(rolehaspermission.PermissionIDIn(permIDs...)).
		Where(applyRoleHasPermissionTenantPredicate(tenantID)).
		Select(rolehaspermission.FieldRoleID).
		Ints(ctx)
	if err != nil {
		return nil, err
	}
	if len(roleIDs) == 0 {
		return nil, nil
	}

	// Step 3: load the roles.
	q := r.client.Role.Query().
		Where(
			role.IDIn(idsToUints(roleIDs)...),
			role.DeletedAtIsNil(),
		)
	q = applyRoleTenantScope(q, tenantID)

	rows, err := q.All(ctx)
	if err != nil {
		return nil, err
	}
	return fromEntRoleSlice(rows), nil
}

func (r *Repository) SyncDirectPermissions(ctx context.Context, modelType string, modelID uint, permissionIDs []uint, tenantID *string) error {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}

	// Remove all existing direct permissions for this model.
	if _, err := tx.ModelHasPermission.Delete().
		Where(
			modelhaspermission.ModelType(modelType),
			modelhaspermission.ModelID(modelID),
		).
		Exec(ctx); err != nil {
		return rollback(tx, err)
	}

	// Insert the new set.
	for _, permissionID := range permissionIDs {
		perm, err := tx.Permission.Query().
			Where(permission.ID(permissionID), permission.DeletedAtIsNil()).
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return rollback(tx, storage.ErrPermissionNotFound)
			}
			return rollback(tx, err)
		}
		if err := validateTenantLink(perm.TenantID, tenantID); err != nil {
			return rollback(tx, err)
		}

		b := tx.ModelHasPermission.Create().
			SetPermissionID(permissionID).
			SetModelType(modelType).
			SetModelID(modelID)
		if perm.TenantID != nil {
			b = b.SetTenantID(*perm.TenantID)
		}
		if _, err := b.Save(ctx); err != nil {
			return rollback(tx, err)
		}
	}
	return tx.Commit()
}

func (r *Repository) SyncPermissionsForRole(ctx context.Context, roleID uint, permissionIDs []uint) error {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}

	if _, err := tx.RoleHasPermission.Delete().
		Where(rolehaspermission.RoleID(roleID)).
		Exec(ctx); err != nil {
		return rollback(tx, err)
	}

	// Load the role once so we can validate each link.
	roleRow, err := tx.Role.Query().
		Where(role.ID(roleID), role.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return rollback(tx, storage.ErrRoleNotFound)
		}
		return rollback(tx, err)
	}

	for _, permissionID := range permissionIDs {
		perm, err := tx.Permission.Query().
			Where(permission.ID(permissionID), permission.DeletedAtIsNil()).
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return rollback(tx, storage.ErrPermissionNotFound)
			}
			return rollback(tx, err)
		}
		if err := validateTenantLink(perm.TenantID, roleRow.TenantID); err != nil {
			return rollback(tx, err)
		}

		b := tx.RoleHasPermission.Create().
			SetPermissionID(permissionID).
			SetRoleID(roleID)
		if perm.TenantID != nil {
			b = b.SetTenantID(*perm.TenantID)
		}
		if _, err := b.Save(ctx); err != nil {
			return rollback(tx, err)
		}
	}
	return tx.Commit()
}

func (r *Repository) RevokeAllPermissionsForRole(ctx context.Context, roleID uint) error {
	_, err := r.client.RoleHasPermission.Delete().
		Where(rolehaspermission.RoleID(roleID)).
		Exec(ctx)
	return err
}

func (r *Repository) RolePermissionExists(ctx context.Context, roleID uint, permissionName, guardName string, tenantID *string) (bool, error) {
	permIDs, err := r.client.RoleHasPermission.Query().
		Where(rolehaspermission.RoleID(roleID)).
		Where(applyRoleHasPermissionTenantPredicate(tenantID)).
		Select(rolehaspermission.FieldPermissionID).
		Ints(ctx)
	if err != nil {
		return false, err
	}
	if len(permIDs) == 0 {
		return false, nil
	}

	exists, err := r.client.Permission.Query().
		Where(
			permission.IDIn(idsToUints(permIDs)...),
			permission.Name(permissionName),
			permission.GuardName(guardName),
			permission.DeletedAtIsNil(),
		).
		Where(applyPermissionTenantScopePredicate(tenantID)).
		Exist(ctx)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func applyPermissionTenantScopePredicate(tenantID *string) predicate.Permission {
	if tenantID == nil || *tenantID == "" {
		return permission.TenantIDIsNil()
	}
	return permission.Or(
		permission.TenantID(*tenantID),
		permission.TenantIDIsNil(),
	)
}

// idsToUints converts Ent's []int query result into []uint, which is
// what .IDIn and .RoleIDIn predicates require for uint columns.
func idsToUints(in []int) []uint {
	out := make([]uint, len(in))
	for i, v := range in {
		out[i] = uint(v)
	}
	return out
}

var _ storage.Repository = (*Repository)(nil)
