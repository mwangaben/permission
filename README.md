# Permission Package

A flexible, Laravel-style permission system for Go with multi-tenant support.
Works with **GORM** and **Ent** out of the box. Inspired by
[Spatie/laravel-permission](https://spatie.be/docs/laravel-permission/v8/introduction).

![Go Version](https://img.shields.io/badge/Go-1.24.2-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)
![Go Report Card](https://goreportcard.com/badge/github.com/mwangaben/permission)
![Tests](https://img.shields.io/badge/tests-61%20passing-brightgreen.svg)

## Features

- ✅ **Role-Based Access Control** — Roles and permissions with a familiar API
- ✅ **Direct Permissions** — Assign permissions to a model without a role
- ✅ **Polymorphic Assignments** — Attach roles/permissions to any model type (`user`, `admin`, …)
- ✅ **Multi-Tenant Support** — Optional strict tenant isolation via `tenant_id`
- ✅ **Guard Support** — Multiple guards (`web`, `api`, `admin`, …)
- ✅ **GORM integration** — MySQL, MariaDB, Postgres, SQLite
- ✅ **Ent integration** — Same tables, same behavior, chosen by the handle you pass in
- ✅ **Backend auto-detection** — `NewManager` figures out which backend to use
- ✅ **Authorization Guard** — `Authorize`, `AuthorizeAny`, `AuthorizeAll`
- ✅ **Full test suite** — 61 specs, Ginkgo/Gomega, `-race` clean, both backends

## Installation

```bash
go get github.com/mwangaben/permission
```

## Storage Backends

The package is storage-agnostic. You pick a backend by passing a database
handle to `NewManager`:

| Backend | Pass to `NewManager` | Driver constant |
|---|---|---|
| GORM | `*gorm.DB` | `storage.DriverGorm` |
| Ent  | `*ent.Client` (generated from `storage/entstore/schema`) | `storage.DriverEnt` |

Both backends share the same SQL schema. You can migrate an existing GORM
database to Ent (or vice versa) without changing any data, provided you keep
the same table names — which the package does for you.

## Quick Start

### 1. Initialize the Permission System

#### With GORM

```go
package main

import (
	"context"

	"github.com/mwangaben/permission/permission"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	dsn := "user:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	pm, err := permission.NewManager(db)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	if err := pm.Migrate(ctx); err != nil {
		panic(err)
	}
	if err := pm.SeedDefaultPermissions(ctx); err != nil {
		panic(err)
	}
	if err := pm.SeedDefaultRoles(ctx); err != nil {
		panic(err)
	}
}
```

#### With Ent

```go
package main

import (
	"context"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/lib/pq"

	"github.com/mwangaben/permission/permission"
	"github.com/mwangaben/permission/storage/entstore/ent"
)

func main() {
	drv, err := entsql.Open(dialect.Postgres, "host=localhost user=... dbname=...")
	if err != nil {
		panic(err)
	}
	client := ent.NewClient(ent.Driver(drv))
	defer client.Close()

	pm, err := permission.NewManager(client)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	if err := pm.Migrate(ctx); err != nil {
		panic(err)
	}
	// ... SeedDefaultPermissions, SeedDefaultRoles, etc.
}
```

### 2. Register Permissions

```go
ctx := context.Background()

// Single permission
perm, err := pm.Registrar.Register(ctx, "user.view", "web")

// Multiple permissions
permissions := []struct{ Name, GuardName string }{
	{"user.view", "web"},
	{"user.create", "web"},
	{"user.update", "web"},
	{"user.delete", "web"},
}
created, err := pm.Registrar.RegisterMany(ctx, permissions)

// Global permission (available to all tenants)
globalPerm, err := pm.Registrar.RegisterGlobal(ctx, "system.view", "web")
```

### 3. Create Roles

```go
roleManager := role.NewManager(pm.Repo, pm.Config, pm.Tenant)
roleObj, err := roleManager.Registrar.Register(ctx, "admin", "web")
```

### 4. Assign Permissions to Roles

```go
// By ID
permManager := role.NewPermissionManager(pm.Repo)
err = permManager.AssignPermissionToRole(ctx, perm.ID, roleObj.ID)

// By name
err = permManager.AssignPermissionToRoleByName(ctx, "user.view", "admin", "web", nil)
```

The last argument is the tenant ID (`*string`). Pass `nil` for global scope,
or a pointer to a tenant ID when running with tenant mode enabled.

### 5. Assign Roles to Models

```go
assigner := role.NewAssigner(pm.Repo)

// By ID
err = assigner.AssignRoleToModel(ctx, roleObj.ID, "user", userID, nil)

// By name
err = assigner.AssignRoleToModelByName(ctx, "admin", "user", userID, "web", nil)
```

The `modelType` is any string you use to identify the model class —
typically `"user"`, `"admin"`, `"team"`, etc. The package treats it as an
opaque discriminator for polymorphic assignment.

### 6. Check Permissions

```go
checker := permission.NewChecker(pm.Repo, pm.Config, pm.Tenant)

// Single permission
has, err := checker.HasPermission(ctx, "user", userID, "user.view", "web")

// Any / All
has, err = checker.HasAnyPermission(ctx, "user", userID, "web", "user.view", "user.delete")
all, err := checker.GetAllPermissionsForModel(ctx, "user", userID, "web")
```

### 7. Authorization Guard

The guard wraps the checker with a clean authorize-or-error API.

```go
guard := permission.NewGuard(checker)

// Requires the permission
if err := guard.Authorize(ctx, "user", userID, "user.update", "web"); err != nil {
	return fmt.Errorf("unauthorized: %w", err)
}

// Requires ANY of the listed permissions
err := guard.AuthorizeAny(ctx, "user", userID, "web", "user.view", "user.edit")

// Requires ALL of the listed permissions
err := guard.AuthorizeAll(ctx, "user", userID, "web", "user.view", "user.edit")
```

### 8. Direct Permissions (Without Roles)

```go
directAssigner := permission.NewDirectAssigner(pm.Repo)

err := directAssigner.AssignPermissionToModel(ctx, perm.ID, "user", userID, nil)

// Or by permission name
err = directAssigner.AssignPermissionToModelByName(ctx, "special.access", "user", userID, "web", nil)

// Check
has, err := directAssigner.HasDirectPermission(ctx, "user", userID, "special.access", "web", nil)
```

### 9. Multi-Tenant Support

```go
// Enable tenant mode at construction time
pm, err := permission.NewManager(db, config.WithTenant("string"))
if err != nil {
	panic(err)
}

// Set the active tenant for the manager
pm.WithTenant("tenant-123")

// Now Register/Find/Assign operations are scoped to tenant-123
perm, err := pm.Registrar.Register(ctx, "tenant.data.view", "web")
// perm.TenantID == pointer to "tenant-123"

// Register a global permission while tenant mode is active
globalPerm, err := pm.Registrar.RegisterGlobal(ctx, "system.view", "web")
// globalPerm.TenantID == nil

// Switch tenant
pm.WithTenant("tenant-456")
```

**Isolation rules:**

- **Reads** return rows where `tenant_id = :current OR tenant_id IS NULL` —
  tenant-specific rows plus global rows.
- **Writes** use the exact tenant ID (no NULL fallback).
- **Cross-tenant assignment** is rejected with `storage.ErrCrossTenantAssignment`.
  You cannot link a permission from tenant A to a role in tenant B.

## Architecture

### Database Schema

All tables are managed by both backends. Pivot tables have a surrogate
`id` primary key so that Ent (which doesn't support composite primary
keys idiomatically) and GORM can share the same schema.

```sql
CREATE TABLE permissions (
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    guard_name    VARCHAR(100) DEFAULT 'web',
    tenant_id     VARCHAR(100) NULL,
    created_at    DATETIME NULL,
    updated_at    DATETIME NULL,
    deleted_at    DATETIME NULL,
    UNIQUE INDEX idx_permissions_name_guard_tenant (name, guard_name, tenant_id)
);

CREATE TABLE roles (
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    guard_name    VARCHAR(100) DEFAULT 'web',
    tenant_id     VARCHAR(100) NULL,
    created_at    DATETIME NULL,
    updated_at    DATETIME NULL,
    deleted_at    DATETIME NULL,
    UNIQUE INDEX idx_roles_name_guard_tenant (name, guard_name, tenant_id)
);

CREATE TABLE role_has_permissions (
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    permission_id BIGINT UNSIGNED NOT NULL,
    role_id       BIGINT UNSIGNED NOT NULL,
    tenant_id     VARCHAR(100) NULL,
    UNIQUE INDEX idx_role_permissions_unique (permission_id, role_id)
);

CREATE TABLE model_has_roles (
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    role_id       BIGINT UNSIGNED NOT NULL,
    model_type    VARCHAR(255) NOT NULL,
    model_id      BIGINT UNSIGNED NOT NULL,
    tenant_id     VARCHAR(100) NULL,
    UNIQUE INDEX idx_model_roles_unique (role_id, model_type, model_id)
);

CREATE TABLE model_has_permissions (
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    permission_id BIGINT UNSIGNED NOT NULL,
    model_type    VARCHAR(255) NOT NULL,
    model_id      BIGINT UNSIGNED NOT NULL,
    tenant_id     VARCHAR(100) NULL,
    UNIQUE INDEX idx_model_permissions_unique (permission_id, model_type, model_id)
);

CREATE TABLE tenants (
    id            VARCHAR(100) PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    slug          VARCHAR(255) UNIQUE NOT NULL,
    -- ...
);

CREATE TABLE tenant_user (
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id       VARCHAR(100) NOT NULL,
    tenant_id     VARCHAR(100) NOT NULL,
    UNIQUE INDEX idx_tenant_user_unique (user_id, tenant_id)
);
```

### Package Structure

```
permission/
├── permission/               # Core: Manager, Registrar, Checker, Guard, DirectAssigner
├── role/                     # Role management, assignment, permission linking
├── tenant/                   # Tenant context manager
├── config/                   # Configuration
├── models/                   # GORM models (used by the gormstore backend)
├── storage/                  # Backend-agnostic storage layer
│   ├── storage.go            #   Repository interface
│   ├── types.go              #   Plain structs (Permission, Role, ...)
│   ├── errors.go             #   Shared error vocabulary
│   ├── detect.go             #   Driver auto-detection
│   ├── gormstore/            #   GORM implementation
│   └── entstore/             #   Ent implementation
│       ├── schema/           #   Ent schemas
│       └── ent/              #   Generated code
└── tests/                    # Ginkgo/Gomega specs, both backends
```

## API Reference

All methods take a `context.Context` as their first argument. Constructors
return `(*T, error)`. Every method takes a `*string` tenant ID where scope
matters (nil for global).

### `permission.Manager`

| Method | Description |
|---|---|
| `NewManager(db interface{}, opts ...func(*config.Config)) (*Manager, error)` | Creates a manager. Accepts `*gorm.DB` or `*ent.Client`. |
| `Migrate(ctx) error` | Runs migrations for the active backend. |
| `SeedDefaultPermissions(ctx) error` | Seeds the built-in permission set. |
| `SeedDefaultRoles(ctx) error` | Seeds the built-in role set with permission assignments. |
| `WithTenant(tenantID string) *Manager` | Sets the active tenant (mutates in place). |
| `EnableTenant(tenantIDType string) *Manager` | Enables tenant mode. |
| `DisableTenant() *Manager` | Disables tenant mode. |

**Fields** (exported): `Repo storage.Repository`, `Config *config.Config`,
`Tenant *tenant.Manager`, `Registrar *Registrar`, `Checker *Checker`,
`Guard *Guard`, `RoleRegistrar *role.Registrar`, `RoleAssigner *role.Assigner`,
`RolePermManager *role.PermissionManager`.

The old `pm.DB` field no longer exists. Use `pm.Repo` (a
`storage.Repository`) for backend-agnostic access, or type-assert to
`*gormstore.Repository` / `*entstore.Repository` if you need the raw handle.

### `permission.Registrar`

| Method | Description |
|---|---|
| `Register(ctx, name, guardName string) (*storage.Permission, error)` | Idempotent create-or-return. |
| `RegisterMany(ctx, permissions []struct{Name, GuardName string}) ([]*storage.Permission, error)` | Batch register. |
| `RegisterGlobal(ctx, name, guardName string) (*storage.Permission, error)` | Creates a global (tenant_id = NULL) permission. |
| `FindByName(ctx, name, guardName string) (*storage.Permission, error)` | Lookup, tenant-scoped. |

### `permission.Checker`

| Method | Description |
|---|---|
| `NewChecker(repo storage.Repository, config *config.Config, tenant *tenant.Manager) *Checker` | Constructs a checker. |
| `HasPermission(ctx, modelType string, modelID uint, permissionName, guardName string) (bool, error)` | Checks direct + role-derived permissions. |
| `HasAnyPermission(ctx, modelType string, modelID uint, guardName string, permissionNames ...string) (bool, error)` | Any-of check. |
| `GetAllPermissionsForModel(ctx, modelType string, modelID uint, guardName string) ([]*storage.Permission, error)` | Merges direct and role-derived permissions, deduped on `(name, guard, tenant)`. |

### `permission.Guard`

| Method | Description |
|---|---|
| `NewGuard(checker *Checker) *Guard` | Constructs a guard. |
| `Authorize(ctx, modelType string, modelID uint, permission, guardName string) error` | Errors if not allowed. |
| `AuthorizeAny(ctx, modelType string, modelID uint, guardName string, permissions ...string) error` | Errors unless any permission is held. |
| `AuthorizeAll(ctx, modelType string, modelID uint, guardName string, permissions ...string) error` | Errors unless all permissions are held. |
| `Allows(...) (bool, error)` / `Denies(...) (bool, error)` | Boolean variants. |

### `role.Manager`

| Method | Description |
|---|---|
| `NewManager(repo storage.Repository, config *config.Config, tenant *tenant.Manager) *Manager` | Constructs a role manager. |
| `WithTenant(tenantID string) *Manager` | Returns a **new** manager scoped to the tenant. |

Fields: `Registrar *Registrar`, `Assigner *Assigner`, `PermManager *PermissionManager`.

### `role.Registrar`

| Method | Description |
|---|---|
| `Register(ctx, name, guardName string) (*storage.Role, error)` | Idempotent create-or-return. |
| `RegisterGlobal(ctx, name, guardName string) (*storage.Role, error)` | Global role. |
| `FindByName(ctx, name, guardName string) (*storage.Role, error)` | Lookup, tenant-scoped. |

### `role.Assigner`

| Method | Description |
|---|---|
| `AssignRoleToModel(ctx, roleID uint, modelType string, modelID uint, tenantID *string) error` | Assign. |
| `AssignRoleToModelByName(ctx, roleName, modelType string, modelID uint, guardName string, tenantID *string) error` | Assign by name. |
| `RemoveRoleFromModel(ctx, roleID uint, modelType string, modelID uint) error` | Unassign. |
| `GetRolesForModel(ctx, modelType string, modelID uint, tenantID *string) ([]*storage.Role, error)` | List. |
| `SyncRolesForModel(ctx, modelType string, modelID uint, roleIDs []uint, tenantID *string) error` | Replace all. |

### `role.PermissionManager`

| Method | Description |
|---|---|
| `AssignPermissionToRole(ctx, permissionID, roleID uint) error` | Assign. |
| `AssignPermissionToRoleByName(ctx, permissionName, roleName, guardName string, tenantID *string) error` | Assign by name. |
| `RemovePermissionFromRole(ctx, permissionID, roleID uint) error` | Unassign. |
| `RemovePermissionFromRoleByName(ctx, permissionName, roleName, guardName string, tenantID *string) error` | Unassign by name. |
| `GetPermissionsForRole(ctx, roleID uint) ([]*storage.Permission, error)` | List. |
| `HasPermissionForRole(ctx, roleID uint, permissionName, guardName string, tenantID *string) (bool, error)` | Check. |
| `SyncPermissionsForRole(ctx, roleID uint, permissionIDs []uint) error` | Replace all. |
| `RevokeAllPermissionsForRole(ctx, roleID uint) error` | Clear. |

### `permission.DirectAssigner`

| Method | Description |
|---|---|
| `AssignPermissionToModel(ctx, permissionID uint, modelType string, modelID uint, tenantID *string) error` | Assign directly. |
| `AssignPermissionToModelByName(ctx, permissionName, modelType string, modelID uint, guardName string, tenantID *string) error` | Assign by name. |
| `RemovePermissionFromModel(ctx, permissionID uint, modelType string, modelID uint) error` | Unassign. |
| `GetDirectPermissionsForModel(ctx, modelType string, modelID uint, guardName string, tenantID *string) ([]*storage.Permission, error)` | List. |
| `HasDirectPermission(ctx, modelType string, modelID uint, permissionName, guardName string, tenantID *string) (bool, error)` | Check. |

## Usage Examples

### Basic Check in a Service

```go
func (s *UserService) CanViewUser(ctx context.Context, userID, targetUserID uint) bool {
	checker := permission.NewChecker(s.pm.Repo, s.pm.Config, s.pm.Tenant)
	has, err := checker.HasPermission(ctx, "user", userID, "user.view", "web")
	if err != nil || !has {
		return false
	}
	return true
}
```

### Guard in a GraphQL Resolver

```go
func (r *Resolver) UpdateUser(ctx context.Context, args UpdateUserArgs) (*UserResolver, error) {
	userID := ctx.Value("user_id").(uint)

	checker := permission.NewChecker(r.pm.Repo, r.pm.Config, r.pm.Tenant)
	guard := permission.NewGuard(checker)

	if err := guard.Authorize(ctx, "user", userID, "user.update", "web"); err != nil {
		return nil, err
	}

	user, err := r.userService.UpdateUser(ctx, args.ID, args.Input)
	return &UserResolver{user: user}, err
}
```

### HTTP Middleware

```go
func RequirePermission(pm *permission.Manager, perm string) func(http.Handler) http.Handler {
	checker := permission.NewChecker(pm.Repo, pm.Config, pm.Tenant)
	guard := permission.NewGuard(checker)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := r.Context().Value("user_id").(uint)
			if err := guard.Authorize(r.Context(), "user", userID, perm, "web"); err != nil {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

## Testing

```bash
# Run all tests against Postgres (GORM + Ent conformance)
go test ./tests/... -race -count=1

# Verbose
go test ./tests/... -race -count=1 -v

# Focus on a specific spec
go test ./tests/... -race -count=1 -args --ginkgo.focus="Ent backend"
```

The suite runs against a real Postgres database. Both backends use the same
environment variables:

```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=permission_test
DB_SSLMODE=disable
```

## License

MIT — see [LICENSE](LICENSE).

## Contributing

1. Fork the repository.
2. Create your feature branch (`git checkout -b feature/amazing-feature`).
3. Commit your changes (`git commit -m 'Add amazing feature'`).
4. Push to the branch (`git push origin feature/amazing-feature`).
5. Open a Pull Request.

## Migration from v1.0

`v1.1.0` introduces a storage abstraction that supports both GORM and
Ent backends. The public API changed in a few mechanical ways. This
section walks through the diffs.

### 1. `NewManager` returns an error

```go
// v1.0
pm := permission.NewManager(db)

// v1.1
pm, err := permission.NewManager(db)
if err != nil {
	panic(err)
}
```

The constructor can fail if it can't detect the database driver.

### 2. Every method takes a `context.Context`

```go
// v1.0
perm, err := pm.Registrar.Register("user.view", "web")
has, err := checker.HasPermission("user", userID, "user.view", "web")

// v1.1
perm, err := pm.Registrar.Register(ctx, "user.view", "web")
has, err := checker.HasPermission(ctx, "user", userID, "user.view", "web")
```

### 3. Constructors take `pm.Repo`, not `db`

```go
// v1.0
checker := permission.NewChecker(db, pm.Config, pm.Tenant)
assigner := role.NewAssigner(db)
permManager := role.NewPermissionManager(db)

// v1.1
checker := permission.NewChecker(pm.Repo, pm.Config, pm.Tenant)
assigner := role.NewAssigner(pm.Repo)
permManager := role.NewPermissionManager(pm.Repo)
```

`pm.Repo` is a `storage.Repository`, so the same code works against
GORM or Ent.

### 4. Assignment methods take a tenant ID

```go
// v1.0
assigner.AssignRoleToModel(roleID, "user", userID)

// v1.1 (global scope)
assigner.AssignRoleToModel(ctx, roleID, "user", userID, nil)

// v1.1 (tenant scope)
assigner.AssignRoleToModel(ctx, roleID, "user", userID, &tenantID)
```

### 5. `pm.DB` is gone

```go
// v1.0
db := pm.DB

// v1.1
repo := pm.Repo
// If you need the raw handle:
if gormRepo, ok := pm.Repo.(*gormstore.Repository); ok {
	db := gormRepo.DB() // you'd need to add this accessor
}
```

For most use cases, `pm.Repo` is what you want — it's what the rest of
the package uses internally.

### 6. Pivot tables gained a surrogate primary key

If you manage your own migrations:

```sql
-- v1.0
ALTER TABLE role_has_permissions
	DROP PRIMARY KEY,
	ADD COLUMN id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY FIRST,
	ADD UNIQUE INDEX idx_role_permissions_unique (permission_id, role_id);

-- Same pattern for model_has_roles, model_has_permissions, tenant_user.
```

If you use `pm.Migrate(ctx)`, this is handled automatically.

### 7. Package types

```go
// v1.0
import "github.com/mwangaben/permission/models"
var perm *models.Permission

// v1.1
import "github.com/mwangaben/permission/storage"
var perm *storage.Permission
```

The `models` package still exists for the GORM backend, but consumers
should use `storage.*` types for backend-agnostic code.

## Credits

Inspired by [Spatie/laravel-permission](https://spatie.be/docs/laravel-permission/v8/introduction).

Built with [GORM](https://gorm.io/) and [Ent](https://entgo.io/).

Testing with [Ginkgo](https://onsi.github.io/ginkgo/) & [Gomega](https://onsi.github.io/gomega/).