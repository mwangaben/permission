package middleware

import (
	"context"
	"net/http"

	"github.com/mwangaben/permission/gate"
)

// AuthorizeMiddleware checks if the user has the required permission
func AuthorizeMiddleware(g *gate.Gate, permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := r.Context().Value("user")
			if user == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if !g.Allows(r.Context(), user, permission) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RoleMiddleware checks if the user has the required role
func RoleMiddleware(g *gate.Gate, role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := r.Context().Value("user")
			if user == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if !g.Allows(r.Context(), user, "role:"+role) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
