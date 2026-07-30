package gate

import "context"

// Ability is a function that determines if a user can perform an action
type Ability func(ctx context.Context, user interface{}, arguments ...interface{}) bool

// AbilityWithResponse is a function that returns a full response
type AbilityWithResponse func(ctx context.Context, user interface{}, arguments ...interface{}) *Response

// AbilityFunc is a helper to create abilities
type AbilityFunc func(ctx context.Context, user interface{}, args ...interface{}) bool

// AbilityWithPermission creates an ability that checks a specific permission
func AbilityWithPermission(permissionChecker func(ctx context.Context, userID string, permission string) bool, permission string) Ability {
	return func(ctx context.Context, user interface{}, args ...interface{}) bool {
		userID := getUserID(user)
		if userID == "" {
			return false
		}
		return permissionChecker(ctx, userID, permission)
	}
}

// getUserID extracts user ID from user object
func getUserID(user interface{}) string {
	// This is a placeholder - your application should implement this
	// based on your user model
	if u, ok := user.(interface{ GetID() string }); ok {
		return u.GetID()
	}
	return ""
}
