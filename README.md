# Permission Package
#### A Laravel-style permission system for Go with multi-tenant support.
#### Inspired by Spatie/laravel-permission.

![Go Version](https://img.shields.io/badge/Go-1.24.2-blue.svg)![License](https://img.shields.io/badge/license-MIT-green.svg)![Go Report Card](https://goreportcard.com/badge/github.com/mwangaben/permission)![Tests](https://img.shields.io/badge/tests-passing-brightgreen.svg)

### Features

- **✅ Model-based Permissions: {model}.{action} format (user.view, post.create)**

- **✅ Role Management: Create roles and assign permissions**

- **✅ Multi-Tenant Support: Tenant-specific permissions and roles**

- **✅ Permission Checking: Simple and efficient permission checks**

- **✅ Gate/Ability System: Define custom authorization rules**

- **✅ Before/After Hooks: Intercept authorization checks**

- **✅ GORM Integration: Works with MariaDB/MySQL**

- **✅ REST & GraphQL Ready: Works with any Go web framework**

- **✅ Full Test Coverage: 100% test coverage**

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

#### 2. Setup Database

```go
import (
    "gorm.io/driver/mysql"
    "gorm.io/gorm"
    "github.com/mwangaben/permission/models"
)

func setupDB() (*gorm.DB, error) {
    dsn := "user:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True"
    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
    if err != nil {
        return nil, err
    }
    
    // Auto migrate the permission tables
    db.AutoMigrate(&models.Permission{}, &models.Role{})
    
    return db, nil
}
```

#### 3. Register Permissions

```go
import (
    "github.com/mwangaben/permission/permission"
    "github.com/mwangaben/permission/models"
)

func registerPermissions(db *gorm.DB) error {
    registrar := permission.NewRegistrar(db)
    
    // Register individual permissions
    perm := &models.Permission{
        Name:        "user.view",
        Model:       "user",
        Action:      "view",
        DisplayName: "View Users",
        Description: "Can view user list and user details",
    }
    if err := registrar.Register(perm); err != nil {
        return err
    }
    
    // Register all CRUD permissions for a model
    return registrar.RegisterModelPermissions("post", "Post", nil)
}
```

#### 4. Create Roles

```go
import (
"github.com/mwangaben/permission/role"
"github.com/mwangaben/permission/models"
)

func createRoles(db *gorm.DB) error {
roleReg := role.NewRegistrar(db)

// Get permissions
var viewPermission models.Permission
db.Where("name = ?", "user.view").First(&viewPermission)

var createPermission models.Permission
db.Where("name = ?", "post.create").First(&createPermission)

// Create admin role with permissions
adminRole := &models.Role{
Name:        "admin",
DisplayName: "Administrator",
Description: "Full system access",
Permissions: []models.Permission{viewPermission, createPermission},
IsDefault:   false,
}
return roleReg.Register(adminRole)
}
```

#### 5. Assign Roles to Users

```go
import "github.com/mwangaben/permission/role"

func assignRole(db *gorm.DB, userID, roleName string) error {
    assigner := role.NewAssigner(db)
    return assigner.AssignByName(userID, roleName, nil)
}
```

#### 6. Check Permissions

##### Using the Gate

```go
import "github.com/mwangaben/permission/gate"

func checkPermission(db *gorm.DB, ctx context.Context, userID string) bool {
    g := gate.NewGate(&gate.Config{DB: db})
    
    // Check if user has permission
    if g.Allows(ctx, userID, "user.view") {
        // User has permission
        return true
    }
    return false
}

```

#### Integration with REST API

###  Example: Gin Framework

```go
package main

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
    "github.com/mwangaben/permission/gate"
    "github.com/mwangaben/permission/middleware"
)

func main() {
    db := setupDB()
    r := gin.Default()
    
    // Create gate
    g := gate.NewGate(&gate.Config{DB: db})
    
    // Public routes
    r.POST("/login", loginHandler)
    r.POST("/register", registerHandler)
    
    // Protected routes with permission middleware
    admin := r.Group("/admin")
    admin.Use(middleware.AuthorizeMiddleware(g, "user.view"))
    {
        admin.GET("/users", getUsersHandler)
        admin.POST("/users", middleware.AuthorizeMiddleware(g, "user.create"), createUserHandler)
        admin.PUT("/users/:id", middleware.AuthorizeMiddleware(g, "user.update"), updateUserHandler)
        admin.DELETE("/users/:id", middleware.AuthorizeMiddleware(g, "user.delete"), deleteUserHandler)
    }
    
    r.Run(":8080")
}

// Example handler with manual permission check
func getUsersHandler(c *gin.Context) {
    g := c.MustGet("gate").(*gate.Gate)
    user := c.MustGet("user").(User)
    
    // Manual permission check
    if !g.Allows(c.Request.Context(), user.ID, "user.view") {
        c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
        return
    }
    
    // ... get users from database
    c.JSON(http.StatusOK, users)
}
```

### Integration with GraphQL
```go
package graphql

import (
    "context"
    "fmt"
    "github.com/mwangaben/permission/gate"
    "github.com/mwangaben/permission/middleware"
)

type Resolver struct {
    gate *gate.Gate
    db   *gorm.DB
}

// Public resolvers (no permission needed)
func (r *Resolver) Users(ctx context.Context, args struct{}) ([]*UserResolver, error) {
    // Get all users (public)
    var users []models.User
    r.db.Find(&users)
    return mapUsers(users), nil
}

// Protected resolvers with permission checks
func (r *Resolver) UpdateUser(ctx context.Context, args struct{ ID string, Input UserInput }) (*UserResolver, error) {
    // Get user from context (set by auth middleware)
    userID, ok := ctx.Value("user_id").(string)
    if !ok {
        return nil, fmt.Errorf("unauthorized")
    }
    
    // Check permission
    if !r.gate.Allows(ctx, userID, "user.update") {
        return nil, fmt.Errorf("unauthorized: missing user.update permission")
    }
    
    // Update user logic
    var user models.User
    if err := r.db.First(&user, "id = ?", args.ID).Error; err != nil {
        return nil, err
    }
    
    // Update fields...
    r.db.Save(&user)
    return &UserResolver{user: &user}, nil
}

// Decorator pattern for clean permission checking
func (r *Resolver) DeleteUser(ctx context.Context, args struct{ ID string }) (bool, error) {
    return middleware.AuthorizeMiddleware(r.gate, "user.delete")(func(ctx context.Context) (interface{}, error) {
        // Delete user logic
        return true, nil
    })(ctx)
}
```


### GraphQL Schema Example
```graphql
type Query {
    # Public - no permission needed
    users: [User!]!
    user(id: ID!): User!
    
    # Protected - requires user.view permission
    me: User!
}

type Mutation {
    # Public - no permission needed
    login(email: String!, password: String!): AuthPayload!
    register(input: RegisterInput!): AuthPayload!
    
    # Protected - requires user.update permission
    updateUser(id: ID!, input: UpdateUserInput!): User!
    
    # Protected - requires user.delete permission
    deleteUser(id: ID!): Boolean!
}
```

### GraphQL with Permission Decorators

```go
package graphql

import (
    "context"
    "github.com/mwangaben/permission/decorator"
)

// Permission decorator for clean resolvers
func (r *Resolver) Me(ctx context.Context) (*UserResolver, error) {
    return decorator.Authorize(r.gate, "user.view")(func(ctx context.Context) (interface{}, error) {
        userID := ctx.Value("user_id").(string)
        var user models.User
        if err := r.db.First(&user, "id = ?", userID).Error; err != nil {
            return nil, err
        }
        return &UserResolver{user: &user}, nil
    })(ctx)
}
```

### Multi-Tenant Support
#### Setup Tenant

```go
import "github.com/mwangaben/permission/tenant"

func setupTenant(db *gorm.DB) error {
    resolver := tenant.NewResolver(db)
    
    // Create a tenant
    tenantObj, err := resolver.CreateTenant(
        "My Org",
        "my-org",
        "myorg.example.com",
        "owner-id",
    )
    if err != nil {
        return err
    }
    
    // Add user to tenant
    return resolver.AddUserToTenant("user-id", tenantObj.ID)
}
```

### Tenant-Specific Permissions
```go
func registerTenantPermissions(db *gorm.DB) error {
    registrar := permission.NewRegistrar(db)
    tenantID := "tenant-1"
    
    // Register tenant-specific permission
    perm := &models.Permission{
        Name:        "tenant.data.view",
        Model:       "tenant-data",
        Action:      "view",
        DisplayName: "View Tenant Data",
        TenantID:    &tenantID,  // Only available for this tenant
    }
    return registrar.Register(perm)
}
```

### Check Tenant Permissions
```go
Check Tenant Permissions

```

## Advanced Usage
### Custom Abilities

```go
func registerCustomAbilities(g *gate.Gate) {
    // Simple ability
    g.RegisterAbility("view-dashboard", func(ctx context.Context, user interface{}, args ...interface{}) bool {
        // Custom logic here
        return user != nil
    })
    
    // Ability with response
    g.RegisterAbilityWithResponse("view-admin", func(ctx context.Context, user interface{}, args ...interface{}) *gate.Response {
        if user == nil {
            return gate.NewDenied("User not authenticated")
        }
        // Check if user is admin
        if user.(User).Role != "admin" {
            return gate.NewDenied("Admin role required")
        }
        return gate.NewAllowed()
    })
}
```

### Before/After Hooks

```go
func setupHooks(g *gate.Gate) {
    // Before hook - runs before all checks
    g.Before(func(ctx context.Context, user interface{}, ability string, args ...interface{}) *gate.Response {
        if user == nil {
            return gate.NewDenied("Authentication required")
        }
        if user.(User).IsSuperAdmin() {
            return gate.NewAllowed() // Super admin bypass
        }
        return nil // Continue to next check
    })
    
    // After hook - runs after all checks
    g.After(func(ctx context.Context, user interface{}, ability string, result bool, args ...interface{}) *gate.Response {
        if !result {
            return gate.NewDeniedWithMessage("Access denied", "You don't have permission")
        }
        return nil
    })
}
```

### Policies

```go
type UserPolicy struct {
    gate.BasePolicy
}

func (p *UserPolicy) View(ctx context.Context, user interface{}, targetUser interface{}) bool {
    // Users can view their own profile
    if user.(User).ID == targetUser.(User).ID {
        return true
    }
    // Admins can view any user
    return user.(User).IsAdmin()
}

func (p *UserPolicy) Update(ctx context.Context, user interface{}, targetUser interface{}) bool {
    // Only admins can update users
    return user.(User).IsAdmin()
}

// Register policy
func registerPolicies(g *gate.Gate) {
    g.RegisterPolicy(&models.User{}, &UserPolicy{})
}

// Use policy
func checkUserPolicy(g *gate.Gate, ctx context.Context, user User, targetUser User) bool {
    return g.Allows(ctx, user.ID, "UserPolicy@View", targetUser)
}
```


## Database Schema

### Permissions Table

```sql
CREATE TABLE permissions (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    model VARCHAR(100) NOT NULL,
    action VARCHAR(50) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    tenant_id VARCHAR(100),
    guard_name VARCHAR(100) DEFAULT 'web',
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    UNIQUE INDEX idx_permission_name_tenant (name, tenant_id)
);
```

### Roles Table
```sql
CREATE TABLE roles (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    tenant_id VARCHAR(100),
    guard_name VARCHAR(100) DEFAULT 'web',
    is_default BOOLEAN DEFAULT FALSE,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    UNIQUE INDEX idx_role_name_tenant (name, tenant_id)
);
```


### Role Permissions (Junction Table)

```sql
CREATE TABLE role_permissions (
    role_id VARCHAR(100),
    permission_id VARCHAR(100),
    PRIMARY KEY (role_id, permission_id)
);  
```

### User Roles (Junction Table)
```sql
CREATE TABLE role_user (
    user_id VARCHAR(100),
    role_id VARCHAR(100),
    PRIMARY KEY (user_id, role_id)
);
```

## Testing

```bash
# Setup test database
make db-setup

# Run all tests
make test

# Run specific tests
go test -v ./tests -run TestPermission

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