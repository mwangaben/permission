# Changelog

All notable changes to this project are documented here.
This project adheres to [Semantic Versioning](https://semver.org/).

## [1.1.0] - 2026-09-13

### Added
- **Ent backend alongside GORM.** `NewManager` now accepts either
  `*gorm.DB` or `*ent.Client` and auto-detects the backend.
- `storage.Repository` interface: a backend-agnostic persistence contract
  with 30+ methods.
- `storage/types.go`: plain data structs used by all backends.
- `storage/errors.go`: shared error vocabulary for both backends.
- `storage/detect.go`: driver detection from a database handle.
- Ent conformance test suite: 12 additional specs exercising the
  high-risk repository surface against Postgres.
- Guard-aware `HasPermissionForRole` on `role.PermissionManager`.

### Changed
- `permission.NewManager` now returns `(*Manager, error)`.
- All public methods now take `context.Context` as the first argument.
- Public APIs now accept and return `storage.*` types instead of
  `models.*` types.
- `Manager.DB *gorm.DB` has been removed. Use `Manager.Repo
  storage.Repository` for backend-agnostic access.
- Role/permission assignment methods now take a `tenantID *string`
  argument for tenant-scoped writes.
- Pivot tables (`role_has_permissions`, `model_has_roles`,
  `model_has_permissions`, `tenant_user`) now have a surrogate `id`
  primary key. Uniqueness is preserved via composite unique indexes.

### Fixed
- `HasPermissionForRole` now filters by guard. Previously the guard was
  ignored.
- `GetAllPermissionsForModel` now deduplicates on
  `(name, guard_name, tenant_id)` instead of `name` alone. Previously
  two permissions with the same name under different guards could
  collapse into one.

### Migration Guide

See the [Migration from v1.0](#migration-from-v10) section in the
README.