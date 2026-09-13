// tests/ent_conformance_test.go
package tests

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mwangaben/permission/permission"
	"github.com/mwangaben/permission/role"
	"github.com/mwangaben/permission/storage/entstore/ent"
)

var _ = Describe("Permission System (Ent backend)", func() {
	var (
		ctx     context.Context
		client  *ent.Client
		cleanup func()
		pm      *permission.Manager
	)

	BeforeEach(func() {
		ctx = context.Background()
		client, cleanup = NewTestDBEnt()

		var err error
		pm, err = permission.NewManager(client)
		Expect(err).ToNot(HaveOccurred())

		// Prove that we're actually running against Ent, not GORM.
		Expect(pm.Repo.Name()).To(Equal("ent"))
	})

	AfterEach(func() {
		if cleanup != nil {
			cleanup()
		}
	})

	// ─── Permission registration ─────────────────────────────────────────

	It("registers and finds a permission", func() {
		perm, err := pm.Registrar.Register(ctx, "user.view", "web")
		Expect(err).ToNot(HaveOccurred())
		Expect(perm.ID).ToNot(BeZero())

		found, err := pm.Registrar.FindByName(ctx, "user.view", "web")
		Expect(err).ToNot(HaveOccurred())
		Expect(found.ID).To(Equal(perm.ID))
	})

	It("idempotently registers the same permission", func() {
		p1, err := pm.Registrar.Register(ctx, "user.view", "web")
		Expect(err).ToNot(HaveOccurred())

		p2, err := pm.Registrar.Register(ctx, "user.view", "web")
		Expect(err).ToNot(HaveOccurred())
		Expect(p2.ID).To(Equal(p1.ID))
	})

	It("registers a global permission with nil tenant", func() {
		perm, err := pm.Registrar.RegisterGlobal(ctx, "system.view", "web")
		Expect(err).ToNot(HaveOccurred())
		Expect(perm.TenantID).To(BeNil())
	})

	// ─── Role ↔ Permission ───────────────────────────────────────────────

	It("assigns and removes permissions from a role", func() {
		perm, err := pm.Registrar.Register(ctx, "user.view", "web")
		Expect(err).ToNot(HaveOccurred())

		rm := role.NewManager(pm.Repo, pm.Config, pm.Tenant)
		roleObj, err := rm.Registrar.Register(ctx, "editor", "web")
		Expect(err).ToNot(HaveOccurred())

		pmr := role.NewPermissionManager(pm.Repo)
		Expect(pmr.AssignPermissionToRole(ctx, perm.ID, roleObj.ID)).To(Succeed())

		has, err := pmr.HasPermissionForRole(ctx, roleObj.ID, "user.view", "web", nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(has).To(BeTrue())

		Expect(pmr.RemovePermissionFromRole(ctx, perm.ID, roleObj.ID)).To(Succeed())

		has, err = pmr.HasPermissionForRole(ctx, roleObj.ID, "user.view", "web", nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(has).To(BeFalse())
	})

	It("syncs permissions for a role", func() {
		perm1, err := pm.Registrar.Register(ctx, "user.view", "web")
		Expect(err).ToNot(HaveOccurred())
		perm2, err := pm.Registrar.Register(ctx, "user.edit", "web")
		Expect(err).ToNot(HaveOccurred())
		perm3, err := pm.Registrar.Register(ctx, "user.delete", "web")
		Expect(err).ToNot(HaveOccurred())

		rm := role.NewManager(pm.Repo, pm.Config, pm.Tenant)
		roleObj, err := rm.Registrar.Register(ctx, "editor", "web")
		Expect(err).ToNot(HaveOccurred())

		pmr := role.NewPermissionManager(pm.Repo)
		Expect(pmr.AssignPermissionToRole(ctx, perm1.ID, roleObj.ID)).To(Succeed())
		Expect(pmr.AssignPermissionToRole(ctx, perm2.ID, roleObj.ID)).To(Succeed())

		Expect(pmr.SyncPermissionsForRole(ctx, roleObj.ID, []uint{perm2.ID, perm3.ID})).To(Succeed())

		perms, err := pmr.GetPermissionsForRole(ctx, roleObj.ID)
		Expect(err).ToNot(HaveOccurred())
		Expect(perms).To(HaveLen(2))

		names := []string{perms[0].Name, perms[1].Name}
		Expect(names).To(ConsistOf("user.edit", "user.delete"))
	})

	// ─── Cross-table checks ──────────────────────────────────────────────

	It("checks permission via role", func() {
		perm, err := pm.Registrar.Register(ctx, "user.view", "web")
		Expect(err).ToNot(HaveOccurred())

		rm := role.NewManager(pm.Repo, pm.Config, pm.Tenant)
		roleObj, err := rm.Registrar.Register(ctx, "editor", "web")
		Expect(err).ToNot(HaveOccurred())

		pmr := role.NewPermissionManager(pm.Repo)
		Expect(pmr.AssignPermissionToRole(ctx, perm.ID, roleObj.ID)).To(Succeed())

		assigner := role.NewAssigner(pm.Repo)
		Expect(assigner.AssignRoleToModel(ctx, roleObj.ID, "user", 1, nil)).To(Succeed())

		checker := permission.NewChecker(pm.Repo, pm.Config, pm.Tenant)
		has, err := checker.HasPermission(ctx, "user", 1, "user.view", "web")
		Expect(err).ToNot(HaveOccurred())
		Expect(has).To(BeTrue())
	})

	It("checks direct permissions", func() {
		perm, err := pm.Registrar.Register(ctx, "special.access", "web")
		Expect(err).ToNot(HaveOccurred())

		da := permission.NewDirectAssigner(pm.Repo)
		Expect(da.AssignPermissionToModel(ctx, perm.ID, "user", 2, nil)).To(Succeed())

		checker := permission.NewChecker(pm.Repo, pm.Config, pm.Tenant)
		has, err := checker.HasPermission(ctx, "user", 2, "special.access", "web")
		Expect(err).ToNot(HaveOccurred())
		Expect(has).To(BeTrue())

		// Negative case
		has, err = checker.HasPermission(ctx, "user", 3, "special.access", "web")
		Expect(err).ToNot(HaveOccurred())
		Expect(has).To(BeFalse())
	})

	It("returns all permissions for a model (direct + role-derived)", func() {
		directPerm, err := pm.Registrar.Register(ctx, "direct.one", "web")
		Expect(err).ToNot(HaveOccurred())
		rolePerm, err := pm.Registrar.Register(ctx, "role.one", "web")
		Expect(err).ToNot(HaveOccurred())

		rm := role.NewManager(pm.Repo, pm.Config, pm.Tenant)
		roleObj, err := rm.Registrar.Register(ctx, "editor", "web")
		Expect(err).ToNot(HaveOccurred())

		pmr := role.NewPermissionManager(pm.Repo)
		Expect(pmr.AssignPermissionToRole(ctx, rolePerm.ID, roleObj.ID)).To(Succeed())

		assigner := role.NewAssigner(pm.Repo)
		Expect(assigner.AssignRoleToModel(ctx, roleObj.ID, "user", 3, nil)).To(Succeed())

		da := permission.NewDirectAssigner(pm.Repo)
		Expect(da.AssignPermissionToModel(ctx, directPerm.ID, "user", 3, nil)).To(Succeed())

		checker := permission.NewChecker(pm.Repo, pm.Config, pm.Tenant)
		all, err := checker.GetAllPermissionsForModel(ctx, "user", 3, "web")
		Expect(err).ToNot(HaveOccurred())
		Expect(all).To(HaveLen(2))
	})

	// ─── Tenant isolation ────────────────────────────────────────────────

	It("isolates tenant-scoped permissions", func() {
		pm.EnableTenant("string")
		pm.WithTenant("tenant-1")

		perm, err := pm.Registrar.Register(ctx, "user.view", "web")
		Expect(err).ToNot(HaveOccurred())
		Expect(perm.TenantID).ToNot(BeNil())
		Expect(*perm.TenantID).To(Equal("tenant-1"))

		rm := role.NewManager(pm.Repo, pm.Config, pm.Tenant)
		roleObj, err := rm.Registrar.Register(ctx, "editor", "web")
		Expect(err).ToNot(HaveOccurred())
		Expect(roleObj.TenantID).ToNot(BeNil())
		Expect(*roleObj.TenantID).To(Equal("tenant-1"))

		pmr := role.NewPermissionManager(pm.Repo)
		Expect(pmr.AssignPermissionToRole(ctx, perm.ID, roleObj.ID)).To(Succeed())

		assigner := role.NewAssigner(pm.Repo)
		Expect(assigner.AssignRoleToModel(ctx, roleObj.ID, "user", 1, roleObj.TenantID)).To(Succeed())

		// Same tenant: has permission
		checker := permission.NewChecker(pm.Repo, pm.Config, pm.Tenant)
		has, err := checker.HasPermission(ctx, "user", 1, "user.view", "web")
		Expect(err).ToNot(HaveOccurred())
		Expect(has).To(BeTrue())

		// Different tenant: no permission
		pm.WithTenant("tenant-2")
		checker = permission.NewChecker(pm.Repo, pm.Config, pm.Tenant)
		has, err = checker.HasPermission(ctx, "user", 1, "user.view", "web")
		Expect(err).ToNot(HaveOccurred())
		Expect(has).To(BeFalse())
	})

	It("rejects cross-tenant permission assignment", func() {
		pm.EnableTenant("string")

		pm.WithTenant("tenant-1")
		perm, err := pm.Registrar.Register(ctx, "user.view", "web")
		Expect(err).ToNot(HaveOccurred())

		pm.WithTenant("tenant-2")
		rm := role.NewManager(pm.Repo, pm.Config, pm.Tenant)
		roleObj, err := rm.Registrar.Register(ctx, "role2", "web")
		Expect(err).ToNot(HaveOccurred())

		pmr := role.NewPermissionManager(pm.Repo)
		err = pmr.AssignPermissionToRole(ctx, perm.ID, roleObj.ID)
		Expect(err).To(HaveOccurred())
	})
})
