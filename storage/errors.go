package storage

import "errors"

var (
	// ─── Lookup errors ────────────────────────────────────────────────
	ErrPermissionNotFound = errors.New("storage: permission not found")
	ErrRoleNotFound       = errors.New("storage: role not found")
	ErrTenantNotFound     = errors.New("storage: tenant not found")

	// ─── Creation errors ──────────────────────────────────────────────
	ErrPermissionExists = errors.New("storage: permission already exists")
	ErrRoleExists       = errors.New("storage: role already exists")
	ErrTenantExists     = errors.New("storage: tenant already exists")

	// ─── Cross-tenant enforcement ─────────────────────────────────────
	//
	// Returned by AssignPermissionToRole, AssignRoleToModel, and
	// AssignPermissionToModel when the two resources belong to different
	// tenants (and neither is global). See Repository documentation for
	// the exact rules.
	//ErrCrossTenantAssignment = errors.New("storage: cannot link resources from different tenants")

	// ─── Generic ──────────────────────────────────────────────────────
	ErrDuplicate = errors.New("storage: duplicate record")
)
