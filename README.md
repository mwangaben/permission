# Permission Package
##### A flexible, Laravel-style permission system for Go with multi-tenant support. Built with GORM and designed to be plug-and-play for any Go application.
#### Inspired by Spatie/laravel-permission.

![Go Version](https://img.shields.io/badge/Go-1.24.2-blue.svg)![License](https://img.shields.io/badge/license-MIT-green.svg)![Go Report Card](https://goreportcard.com/badge/github.com/mwangaben/permission)![Tests](https://img.shields.io/badge/tests-passing-brightgreen.svg)

### Features

- **✅ Polymorphic Relationships - Assign permissions and roles to any model (users, admins, etc.)**

- **✅ Role-Based Access Control (RBAC) - Create roles and assign permissions**

- **✅ Direct Permissions - Assign permissions directly to models without roles**

- **✅ Multi-Tenant Support - Optional tenant isolation with tenant_id scoping**

- **✅ Guard Support - Multiple guard support (web, api, admin)**

- **✅ GORM Integration - Works seamlessly with GORM and MariaDB/MySQL**

- **✅ Permission Checking - Simple and efficient permission checks**

- **✅ Authorization Guard - Clean authorization API with Authorize, AuthorizeAll, AuthorizeAny**

- **✅ Full Test Coverage - Comprehensive tests with Ginkgo/Gomega**

- **✅ Zero Dependencies - Only depends on GORMchecks**

- **✅ GORM Integration: Works with MariaDB/MySQL**

- **✅ REST & GraphQL Ready: Works with any Go web framework**


### Installation

```bash
go get github.com/mwangaben/permission
```

### Quick Start

#### 1. Define Models

```go
import "github.com/mwangaben/permission/models"

// User model (your existing user model)
type User struct {
    ID       string `gorm:"primaryKey"`
    Name     string
    Email    string
    // ... other fields
}

// You don't need to modify your User model
// The package uses the user ID for permission checks
```

#### 2. Initialize the Permission System

```go
package main

import (
	"github.com/mwangaben/permission/permission"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// Connect to database
	dsn := "user:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	// Create permission manager
	pm := permission.NewManager(db)

	// Run migrations
	if err := pm.Migrate(); err != nil {
		panic(err)
	}

	// Seed default permissions and roles
	if err := pm.SeedDefaultPermissions(); err != nil {
		panic(err)
	}
	if err := pm.SeedDefaultRoles(); err != nil {
		panic(err)
	}
}
```

#### 3. Register Permissions

```go
// Register a single permission
perm, err := pm.Registrar.Register("user.view", "web")

// Register multiple permissions
permissions := []struct{ Name, GuardName string }{
{"user.view", "web"},
{"user.create", "web"},
{"user.update", "web"},
{"user.delete", "web"},
}
created, err := pm.Registrar.RegisterMany(permissions)
```

#### 4. Create Roles

```go
// Create a role
roleManager := role.NewManager(db, pm.Config, pm.Tenant)
roleObj, err := roleManager.Registrar.Register("admin", "web")
```

#### 5. Assign Permissions to Roles

```go
// Get permission
perm, err := pm.Registrar.FindByName("user.view", "web")

// Assign to role
permManager := role.NewPermissionManager(db)
err = permManager.AssignPermissionToRole(perm.ID, roleObj.ID)
```

#### 6. Assign Roles to Users


```go
// Assign role to user
assigner := role.NewAssigner(db)
err = assigner.AssignRoleToModel(roleObj.ID, "user", userID)
```


#### 6. Check Permissions

```go
// Check if user has permission
checker := permission.NewChecker(db, pm.Config, pm.Tenant)
has, err := checker.HasPermission("user", userID, "user.view", "web")

// Or use the guard
guard := permission.NewGuard(checker)
err = guard.Authorize(ctx, "user", userID, "user.view", "web")
```
## Architecture

### Database Schema

```sql
-- Permissions table
CREATE TABLE permissions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    guard_name VARCHAR(100) DEFAULT 'web',
    tenant_id VARCHAR(100) NULL,
    created_at DATETIME NULL,
    updated_at DATETIME NULL,
    deleted_at DATETIME NULL,
    UNIQUE INDEX idx_permissions_name_guard_tenant (name, guard_name, tenant_id)
);

-- Roles table
CREATE TABLE roles (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    guard_name VARCHAR(100) DEFAULT 'web',
    tenant_id VARCHAR(100) NULL,
    created_at DATETIME NULL,
    updated_at DATETIME NULL,
    deleted_at DATETIME NULL,
    UNIQUE INDEX idx_roles_name_guard_tenant (name, guard_name, tenant_id)
);

-- Role has permissions (many-to-many)
CREATE TABLE role_has_permissions (
    permission_id BIGINT UNSIGNED NOT NULL,
    role_id BIGINT UNSIGNED NOT NULL,
    tenant_id VARCHAR(100) NULL,
    PRIMARY KEY (permission_id, role_id),
    FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE
);

-- Model has roles (polymorphic)
CREATE TABLE model_has_roles (
    role_id BIGINT UNSIGNED NOT NULL,
    model_type VARCHAR(255) NOT NULL,
    model_id BIGINT UNSIGNED NOT NULL,
    tenant_id VARCHAR(100) NULL,
    PRIMARY KEY (role_id, model_id, model_type),
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE
);

-- Model has permissions (polymorphic - direct permissions)
CREATE TABLE model_has_permissions (
    permission_id BIGINT UNSIGNED NOT NULL,
    model_type VARCHAR(255) NOT NULL,
    model_id BIGINT UNSIGNED NOT NULL,
    tenant_id VARCHAR(100) NULL,
    PRIMARY KEY (permission_id, model_id, model_type),
    FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
);
```


### Package Structure

```text
permission/
├── permission/
│   ├── manager.go           # Main entry point
│   ├── registrar.go         # Permission registration
│   ├── checker.go           # Permission checking
│   ├── guard.go             # Authorization guard
│   └── direct_assigner.go   # Direct model permissions
├── role/
│   ├── manager.go           # Role management
│   ├── registrar.go         # Role registration
│   ├── assigner.go          # Role assignment to models
│   └── permission_manager.go # Role-permission assignments
├── models/
│   ├── permission.go
│   ├── role.go
│   ├── role_has_permissions.go
│   ├── model_has_roles.go
│   └── model_has_permissions.go
├── config/
│   └── config.go            # Configuration
├── tenant/
│   └── manager.go           # Tenant management
└── tests/
    └── ...                  # Comprehensive tests
```


## Usage Examples

#### Basic Permission Check
```go
func (s *UserService) CanViewUser(ctx context.Context, userID, targetUserID uint) bool {
    pm := s.getPermissionManager() // Get your manager instance
    
    checker := permission.NewChecker(pm.DB, pm.Config, pm.Tenant)
    has, err := checker.HasPermission("user", userID, "user.view", "web")
    if err != nil || !has {
        return false
    }
    return true
}
```

#### Using the Guard for Authorization

```go
func (s *UserService) UpdateUser(ctx context.Context, userID, targetUserID uint, input UpdateUserInput) error {
    pm := s.getPermissionManager()
    checker := permission.NewChecker(pm.DB, pm.Config, pm.Tenant)
    guard := permission.NewGuard(checker)
    
    // Check if user has permission
    err := guard.Authorize(ctx, "user", userID, "user.update", "web")
    if err != nil {
        return fmt.Errorf("unauthorized: %w", err)
    }
    
    // Proceed with update...
    return nil
}
```

#### Checking Multiple Permissions

```go
// Check if user has ANY of the permissions
err := guard.AuthorizeAny(ctx, "user", userID, "web", "user.view", "user.edit", "user.delete")

// Check if user has ALL of the permissions
err := guard.AuthorizeAll(ctx, "user", userID, "web", "user.view", "user.edit")
```


#### Direct Permission Assignment

```go
// Assign permission directly to a user (without role)
directAssigner := permission.NewDirectAssigner(db)
err := directAssigner.AssignPermissionToModel(perm.ID, "user", userID)

// Check direct permission
has, err := directAssigner.HasDirectPermission("user", userID, "special.access", "web")
```

#### Multi-Tenant Support
```go
// Enable tenant mode
pm := permission.NewManager(db, config.WithTenant("string"))

// Set tenant context for all operations
pm.WithTenant("tenant-123")

// Register tenant-specific permission
perm, err := pm.Registrar.Register("tenant.data.view", "web")
// perm.TenantID will be "tenant-123"

// Register global permission (available to all tenants)
globalPerm, err := pm.Registrar.RegisterGlobal("system.view", "web")
// globalPerm.TenantID will be nil

// Switch tenant context
pm.WithTenant("tenant-456")
// Now all operations are scoped to tenant-456
```

### Integration with GraphQL Resolver
```go
func (r *Resolver) UpdateUser(ctx context.Context, args UpdateUserArgs) (*UserResolver, error) {
    // Get current user from context
    userID := ctx.Value("user_id").(uint)
    
    // Check permission using guard
    checker := permission.NewChecker(r.db, r.pm.Config, r.pm.Tenant)
    guard := permission.NewGuard(checker)
    
    if err := guard.Authorize(ctx, "user", userID, "user.update", "web"); err != nil {
        return nil, err
    }
    
    // Proceed with update...
    user, err := r.userService.UpdateUser(ctx, args.ID, args.Input)
    return &UserResolver{user: user}, err
}
```

### API Reference
### Permission Manager (permission.Manager)


| Method                                                               | Description                    |
|----------------------------------------------------------------------|--------------------------------|
| ```NewManager(db *gorm.DB, opts ...func(*config.Config)) *Manager``` | Creates a new manager instance |
| ```Migrate() error```	                                               | Runs database migrations       |
| ```SeedDefaultPermissions() error```	                                | Seeds default permissions      |
| ```SeedDefaultRoles() error```	                                      | Seeds default roles            |
| ```WithTenant(tenantID string) *Manager```	                          | Sets tenant context            |
| ```EnableTenant(tenantIDType string) *Manager```                     | Enables tenant mode            |
| ```DisableTenant() *Manager```	                                      | Disables tenant mode           |

### Registrar (permission.Registrar)
| Method                                                                           | Description                    |
|----------------------------------------------------------------------------------|--------------------------------|
| ``` Register(name, guardName string) (*Permission, error) ```	                   | Registers a new permission     |
| 	```RegisterMany(permissions []struct{Name, GuardName}) ([]Permission, error)``` | Registers multiple permissions |
| ```RegisterGlobal(name, guardName string) (*Permission, error)```	               | Registers a global permission  |
| ```FindByName(name, guardName string) (*Permission, error)```	                   | Finds a permission by name     |


### Checker (permission.Checker)
| Method                                                                                                            | Description                                  |
|-------------------------------------------------------------------------------------------------------------------|----------------------------------------------|
| ```HasPermission(modelType string, modelID uint, permissionName, guardName string) (bool, error)```               | Checks if a model has a permission           |
| ```HasAnyPermission(modelType string, modelID uint, guardName string, permissionNames ...string) (bool, error)``` | Checks if a model has any of the permissions |
| ```GetAllPermissionsForModel(modelType string, modelID uint, guardName string) ([]Permission, error)```           | Gets all permissions for a model             |

### Guard (permission.Guard)

| Method                                                                                                                 | Description                            |
|------------------------------------------------------------------------------------------------------------------------|----------------------------------------|
| ```Authorize(ctx context.Context, modelType string, modelID uint, permission, guardName string) error```               | Authorizes a permission check          |
| ```AuthorizeAny(ctx context.Context, modelType string, modelID uint, guardName string, permissions ...string) error``` | Authorizes if user has any permission  |
| ```AuthorizeAll(ctx context.Context, modelType string, modelID uint, guardName string, permissions ...string) error``` | Authorizes if user has all permissions |

### Role Manager (role.Manager)

| Method                                                                                | Description                   |
|---------------------------------------------------------------------------------------|-------------------------------|
| ```NewManager(db *gorm.DB, config *config.Config, tenant *tenant.Manager) *Manager``` | Creates a new role manager    |
| ```WithTenant(tenantID string) *Manager```                                            | Sets tenant context for roles |
| ```Register(name, guardName string) (*Role, error)```                                 | Registers a new role          |
| ```FindByName(name, guardName string) (*Role, error)	```                              | Finds a role by name          |


### Role Assigner (role.Assigner)

| Method                                                                                          | Description                 |
|-------------------------------------------------------------------------------------------------|-----------------------------|
| ```AssignRoleToModel(roleID uint, modelType string, modelID uint) error```                      | Assigns a role to a model   |
| ```AssignRoleToModelByName(roleName, modelType string, modelID uint, guardName string) error``` | Assigns a role by name      |
| ```RemoveRoleFromModel(roleID uint, modelType string, modelID uint) error```                    | Removes a role from a model |
| ```GetRolesForModel(modelType string, modelID uint) ([]Role, error)```                          | Gets all roles for a model  |
| ```SyncRolesForModel(modelType string, modelID uint, roleIDs []uint) error```                   | Syncs roles for a model     |


### Role Permission Manager (role.PermissionManager)

| Method                                                                                         | Description                         |
|------------------------------------------------------------------------------------------------|-------------------------------------|
| ```AssignPermissionToRole(permissionID, roleID uint) error```                                  | Assigns a permission to a role      |
| ```AssignPermissionToRoleByName(permissionName, roleName, guardName string) error```           | Assigns a permission by name        |
| ```RemovePermissionFromRole(permissionID, roleID uint) error```                                | Removes a permission from a role    |
| ```GetPermissionsForRole(roleID uint) ([]Permission, error)```                                 | Gets all permissions for a role     |
| ```HasPermissionForRole(roleID uint, permissionName string, tenantID *string) (bool, error)``` | Checks if a role has a permission   |
| ```SyncPermissionsForRole(roleID uint, permissionIDs []uint) error```                          | Syncs permissions for a role        |
| ```RevokeAllPermissionsForRole(roleID uint) error```                                           | Revokes all permissions from a role |


## Testing
```bash
# Setup test database
make db-setup

# Run all tests
make test

# Run specific test
make test-specific TEST=TestPermissionRegistration

# Run with Ginkgo
ginkgo -v ./tests

# Run specific suite
ginkgo -v ./tests --focus="Permission System"

# Run with coverage
make test-cover
```



### Configuration
### Environment Variables

```bash
# Database configuration for tests
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=root
DB_NAME=permission_test

```


### License
MIT License - see the LICENSE file for details.


### Contributing
1. Fork the repository

2. Create your feature branch (git checkout -b feature/amazing-feature)

3. Commit your changes (git commit -m 'Add amazing feature')

4. Push to the branch (git push origin feature/amazing-feature)

5. Open a Pull Request


## Credits
Inspired by [Spatie/laravel-permission](https://spatie.be/docs/laravel-permission/v8/introduction)

Built with [GORM](https://gorm.io/)

Testing with [Ginkgo](https://onsi.github.io/ginkgo/) & [Gomega](https://onsi.github.io/gomega/)