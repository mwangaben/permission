package gate

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/mwangaben/permission/models"
	"gorm.io/gorm"
)

// Gate is the main authorization manager
type Gate struct {
	db                    *gorm.DB
	policies              map[string]Policy
	abilities             map[string]Ability
	abilitiesWithResponse map[string]AbilityWithResponse
	beforeHooks           []BeforeHook
	afterHooks            []AfterHook
	tenantID              *string
}

// BeforeHook is a function that runs before ability checks
type BeforeHook func(ctx context.Context, user interface{}, ability string, arguments ...interface{}) *Response

// AfterHook is a function that runs after ability checks
type AfterHook func(ctx context.Context, user interface{}, ability string, result bool, arguments ...interface{}) *Response

// Policy defines the interface for authorization policies
type Policy interface {
	Before(ctx context.Context, user interface{}, ability string, arguments ...interface{}) *Response
	After(ctx context.Context, user interface{}, ability string, result bool, arguments ...interface{}) *Response
}

// Config holds gate configuration
type Config struct {
	DB       *gorm.DB
	TenantID *string
}

// NewGate creates a new Gate instance
func NewGate(config *Config) *Gate {
	if config.DB != nil {
		config.DB.AutoMigrate(&models.Permission{}, &models.Role{})
	}

	return &Gate{
		db:                    config.DB,
		policies:              make(map[string]Policy),
		abilities:             make(map[string]Ability),
		abilitiesWithResponse: make(map[string]AbilityWithResponse),
		tenantID:              config.TenantID,
	}
}

// WithTenant sets the tenant context for the gate
func (g *Gate) WithTenant(tenantID string) *Gate {
	return &Gate{
		db:                    g.db,
		policies:              g.policies,
		abilities:             g.abilities,
		abilitiesWithResponse: g.abilitiesWithResponse,
		beforeHooks:           g.beforeHooks,
		afterHooks:            g.afterHooks,
		tenantID:              &tenantID,
	}
}

// RegisterAbility registers a simple ability
func (g *Gate) RegisterAbility(name string, ability Ability) {
	g.abilities[name] = ability
}

// RegisterAbilityWithResponse registers an ability that returns a full response
func (g *Gate) RegisterAbilityWithResponse(name string, ability AbilityWithResponse) {
	g.abilitiesWithResponse[name] = ability
}

// RegisterPolicy registers a policy
func (g *Gate) RegisterPolicy(model interface{}, policy Policy) {
	modelType := reflect.TypeOf(model)
	key := modelType.String()
	g.policies[key] = policy
}

// Before registers a before hook
func (g *Gate) Before(hook BeforeHook) {
	g.beforeHooks = append(g.beforeHooks, hook)
}

// After registers an after hook
func (g *Gate) After(hook AfterHook) {
	g.afterHooks = append(g.afterHooks, hook)
}

// Allows checks if a user can perform an ability
func (g *Gate) Allows(ctx context.Context, user interface{}, ability string, arguments ...interface{}) bool {
	return g.check(ctx, user, ability, arguments...).Allowed
}

// Denies checks if a user cannot perform an ability
func (g *Gate) Denies(ctx context.Context, user interface{}, ability string, arguments ...interface{}) bool {
	return !g.check(ctx, user, ability, arguments...).Allowed
}

// Check performs an authorization check and returns a full response
func (g *Gate) Check(ctx context.Context, user interface{}, ability string, arguments ...interface{}) *Response {
	return g.check(ctx, user, ability, arguments...)
}

// check performs the actual authorization check
func (g *Gate) check(ctx context.Context, user interface{}, ability string, arguments ...interface{}) *Response {
	// Run before hooks
	for _, hook := range g.beforeHooks {
		if response := hook(ctx, user, ability, arguments...); response != nil {
			return response
		}
	}

	// Check if it's a policy-based ability
	if strings.Contains(ability, "@") {
		parts := strings.Split(ability, "@")
		if len(parts) == 2 {
			policyName := parts[0]
			methodName := parts[1]

			if policy, ok := g.policies[policyName]; ok {
				method := reflect.ValueOf(policy).MethodByName(methodName)
				if method.IsValid() {
					args := []reflect.Value{
						reflect.ValueOf(ctx),
						reflect.ValueOf(user),
					}
					for _, arg := range arguments {
						args = append(args, reflect.ValueOf(arg))
					}

					result := method.Call(args)
					if len(result) > 0 {
						if allowed, ok := result[0].Interface().(bool); ok {
							response := &Response{Allowed: allowed}
							if len(result) > 1 {
								if err, ok := result[1].Interface().(error); ok {
									response.Reason = err.Error()
								}
							}
							return response
						}
						if resp, ok := result[0].Interface().(*Response); ok {
							return resp
						}
					}
				}
			}
		}
	}

	// Check if it's an ability with response
	if abilityFunc, ok := g.abilitiesWithResponse[ability]; ok {
		response := abilityFunc(ctx, user, arguments...)
		if response != nil {
			return response
		}
		return &Response{Allowed: false, Reason: "ability returned nil response"}
	}

	// Check if it's a registered ability
	abilityFunc, abilityExists := g.abilities[ability]
	if abilityExists {
		result := abilityFunc(ctx, user, arguments...)
		response := &Response{Allowed: result}

		// Run after hooks
		for _, hook := range g.afterHooks {
			if afterResponse := hook(ctx, user, ability, result, arguments...); afterResponse != nil {
				return afterResponse
			}
		}

		return response
	}

	// If not found, deny
	return &Response{Allowed: false, Reason: fmt.Sprintf("ability '%s' not found", ability)}
}

// Can checks if a user can perform an ability (alias for Allows)
func (g *Gate) Can(ctx context.Context, user interface{}, ability string, arguments ...interface{}) bool {
	return g.Allows(ctx, user, ability, arguments...)
}

// Cannot checks if a user cannot perform an ability (alias for Denies)
func (g *Gate) Cannot(ctx context.Context, user interface{}, ability string, arguments ...interface{}) bool {
	return g.Denies(ctx, user, ability, arguments...)
}

// GetUserPermissions returns all permissions for a user
func (g *Gate) GetUserPermissions(ctx context.Context, userID string, tenantID *string) ([]models.Permission, error) {
	var permissions []models.Permission

	// Use Unscoped to avoid soft delete on join table
	query := g.db.Unscoped().Table("role_user").
		Select("permissions.*").
		Joins("JOIN roles ON roles.id = role_user.role_id").
		Joins("JOIN role_permissions ON role_permissions.role_id = roles.id").
		Joins("JOIN permissions ON permissions.id = role_permissions.permission_id").
		Where("role_user.user_id = ? AND roles.deleted_at IS NULL", userID)

	if tenantID != nil && *tenantID != "" {
		query = query.Where("(roles.tenant_id = ? OR roles.tenant_id IS NULL)", *tenantID)
		query = query.Where("(permissions.tenant_id = ? OR permissions.tenant_id IS NULL)", *tenantID)
	}

	if err := query.Find(&permissions).Error; err != nil {
		return nil, err
	}

	return permissions, nil
}

// GetUserRoles returns all roles for a user
func (g *Gate) GetUserRoles(ctx context.Context, userID string, tenantID *string) ([]models.Role, error) {
	var roles []models.Role

	// Use Unscoped to avoid soft delete on join table
	query := g.db.Unscoped().Table("role_user").
		Select("roles.*").
		Joins("JOIN roles ON roles.id = role_user.role_id").
		Where("role_user.user_id = ? AND roles.deleted_at IS NULL", userID)

	if tenantID != nil && *tenantID != "" {
		query = query.Where("(roles.tenant_id = ? OR roles.tenant_id IS NULL)", *tenantID)
	}

	if err := query.Find(&roles).Error; err != nil {
		return nil, err
	}

	return roles, nil
}

// HasPermission checks if a user has a specific permission
func (g *Gate) HasPermission(ctx context.Context, userID string, permissionName string, tenantID *string) (bool, error) {
	permissions, err := g.GetUserPermissions(ctx, userID, tenantID)
	if err != nil {
		return false, err
	}

	for _, p := range permissions {
		if p.Name == permissionName {
			return true, nil
		}
	}
	return false, nil
}

// HasModelPermission checks if a user has a model permission
func (g *Gate) HasModelPermission(ctx context.Context, userID, model, action string, tenantID *string) (bool, error) {
	return g.HasPermission(ctx, userID, model+"."+action, tenantID)
}

// HasRole checks if a user has a specific role
func (g *Gate) HasRole(ctx context.Context, userID string, roleName string, tenantID *string) (bool, error) {
	roles, err := g.GetUserRoles(ctx, userID, tenantID)
	if err != nil {
		return false, err
	}

	for _, r := range roles {
		if r.Name == roleName {
			return true, nil
		}
	}
	return false, nil
}
