package tests

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mwangaben/permission/config"
	"github.com/mwangaben/permission/permission"
	"github.com/mwangaben/permission/role"
	"github.com/mwangaben/permission/storage"
	"gorm.io/gorm"
)

var _ = Describe("Role Management", func() {
	var (
		db      *gorm.DB
		cleanup func()
		pm      *permission.Manager
		ctx     context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		db, cleanup = NewTestDBPost()

		var err error
		pm, err = permission.NewManager(db)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if cleanup != nil {
			cleanup()
		}
	})

	Context("Role Registration", func() {
		BeforeEach(func() {
			_, err := pm.Registrar.Register(ctx, "user.view", "web")
			Expect(err).ToNot(HaveOccurred())
			_, err = pm.Registrar.Register(ctx, "user.edit", "web")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should register a new role", func() {
			roleManager := role.NewManager(pm.Repo, pm.Config, pm.Tenant)
			roleObj, err := roleManager.Registrar.Register(ctx, "editor", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(roleObj.Name).To(Equal("editor"))
			Expect(roleObj.GuardName).To(Equal("web"))
		})

		It("should return existing role when registering duplicate", func() {
			roleManager := role.NewManager(pm.Repo, pm.Config, pm.Tenant)
			_, err := roleManager.Registrar.Register(ctx, "editor", "web")
			Expect(err).ToNot(HaveOccurred())

			roleObj, err := roleManager.Registrar.Register(ctx, "editor", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(roleObj.Name).To(Equal("editor"))
		})

		It("should find a role by name", func() {
			roleManager := role.NewManager(pm.Repo, pm.Config, pm.Tenant)
			_, err := roleManager.Registrar.Register(ctx, "editor", "web")
			Expect(err).ToNot(HaveOccurred())

			found, err := roleManager.Registrar.FindByName(ctx, "editor", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(found.Name).To(Equal("editor"))
		})

		It("should register tenant-specific role", func() {
			pm.EnableTenant("string")
			pm.WithTenant("tenant-1")

			roleManager := role.NewManager(pm.Repo, pm.Config, pm.Tenant)
			roleObj, err := roleManager.Registrar.Register(ctx, "tenant-admin", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(roleObj.TenantID).ToNot(BeNil())
			Expect(*roleObj.TenantID).To(Equal("tenant-1"))
		})
	})

	Context("Role Permission Assignment", func() {
		var (
			perm1       *storage.Permission
			perm2       *storage.Permission
			roleObj     *storage.Role
			roleManager *role.Manager
		)

		BeforeEach(func() {
			var err error
			perm1, err = pm.Registrar.Register(ctx, "user.view", "web")
			Expect(err).ToNot(HaveOccurred())
			perm2, err = pm.Registrar.Register(ctx, "user.edit", "web")
			Expect(err).ToNot(HaveOccurred())

			roleManager = role.NewManager(pm.Repo, pm.Config, pm.Tenant)
			roleObj, err = roleManager.Registrar.Register(ctx, "editor", "web")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should assign permissions to a role", func() {
			permManager := role.NewPermissionManager(pm.Repo)
			Expect(permManager.AssignPermissionToRole(ctx, perm1.ID, roleObj.ID)).To(Succeed())
			Expect(permManager.AssignPermissionToRole(ctx, perm2.ID, roleObj.ID)).To(Succeed())

			perms, err := permManager.GetPermissionsForRole(ctx, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(perms).To(HaveLen(2))

			// Instead of reloading the role with preloaded permissions (which
			// storage.Role doesn't carry), check via the repository.
			has, err := permManager.HasPermissionForRole(ctx, roleObj.ID, "user.view", "web", nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeTrue())

			has, err = permManager.HasPermissionForRole(ctx, roleObj.ID, "user.edit", "web", nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeTrue())
		})
	})

	Context("Role Assignment to Models", func() {
		var (
			roleObj  *storage.Role
			assigner *role.Assigner
		)

		BeforeEach(func() {
			var err error
			perm1, err := pm.Registrar.Register(ctx, "user.view", "web")
			Expect(err).ToNot(HaveOccurred())

			roleManager := role.NewManager(pm.Repo, pm.Config, pm.Tenant)
			roleObj, err = roleManager.Registrar.Register(ctx, "viewer", "web")
			Expect(err).ToNot(HaveOccurred())

			permManager := role.NewPermissionManager(pm.Repo)
			Expect(permManager.AssignPermissionToRole(ctx, perm1.ID, roleObj.ID)).To(Succeed())

			assigner = role.NewAssigner(pm.Repo)
		})

		It("should assign role to model", func() {
			Expect(assigner.AssignRoleToModel(ctx, roleObj.ID, "user", 1, nil)).To(Succeed())

			roles, err := assigner.GetRolesForModel(ctx, "user", 1, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(roles).ToNot(BeEmpty())
		})

		It("should assign role to model by name", func() {
			Expect(assigner.AssignRoleToModelByName(ctx, "viewer", "user", 2, "web", nil)).To(Succeed())

			roles, err := assigner.GetRolesForModel(ctx, "user", 2, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(roles).ToNot(BeEmpty())
		})

		It("should remove role from model", func() {
			Expect(assigner.AssignRoleToModel(ctx, roleObj.ID, "user", 1, nil)).To(Succeed())
			Expect(assigner.RemoveRoleFromModel(ctx, roleObj.ID, "user", 1)).To(Succeed())

			roles, err := assigner.GetRolesForModel(ctx, "user", 1, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(roles).To(BeEmpty())
		})

		It("should sync roles for model", func() {
			roleManager := role.NewManager(pm.Repo, pm.Config, pm.Tenant)
			role2, err := roleManager.Registrar.Register(ctx, "editor", "web")
			Expect(err).ToNot(HaveOccurred())

			Expect(assigner.AssignRoleToModel(ctx, roleObj.ID, "user", 3, nil)).To(Succeed())
			Expect(assigner.AssignRoleToModel(ctx, role2.ID, "user", 3, nil)).To(Succeed())

			Expect(assigner.SyncRolesForModel(ctx, "user", 3, []uint{role2.ID}, nil)).To(Succeed())

			roles, err := assigner.GetRolesForModel(ctx, "user", 3, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(roles).To(HaveLen(1))
			Expect(roles[0].ID).To(Equal(role2.ID))
		})
	})

	Context("Role with Tenant Support", func() {
		var tenantPM *permission.Manager

		BeforeEach(func() {
			var err error
			tenantPM, err = permission.NewManager(db, config.WithTenant("string"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should create tenant-specific roles and permissions", func() {
			tenantPM.WithTenant("tenant-1")

			perm1, err := tenantPM.Registrar.Register(ctx, "user.view", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(perm1.TenantID).ToNot(BeNil())
			Expect(*perm1.TenantID).To(Equal("tenant-1"))

			roleManager := role.NewManager(tenantPM.Repo, tenantPM.Config, tenantPM.Tenant)
			role1, err := roleManager.Registrar.Register(ctx, "admin", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(role1.TenantID).ToNot(BeNil())
			Expect(*role1.TenantID).To(Equal("tenant-1"))

			permManager := role.NewPermissionManager(tenantPM.Repo)
			Expect(permManager.AssignPermissionToRole(ctx, perm1.ID, role1.ID)).To(Succeed())

			assigner := role.NewAssigner(tenantPM.Repo)
			Expect(assigner.AssignRoleToModel(ctx, role1.ID, "user", 1, role1.TenantID)).To(Succeed())

			checker := permission.NewChecker(tenantPM.Repo, tenantPM.Config, tenantPM.Tenant)
			has, err := checker.HasPermission(ctx, "user", 1, "user.view", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeTrue())

			tenantPM.WithTenant("tenant-2")
			checker = permission.NewChecker(tenantPM.Repo, tenantPM.Config, tenantPM.Tenant)
			has, err = checker.HasPermission(ctx, "user", 1, "user.view", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeFalse())
		})

		It("should create tenant-specific roles for different tenants", func() {
			roleManager := role.NewManager(tenantPM.Repo, tenantPM.Config, tenantPM.Tenant)

			tenantPM.WithTenant("tenant-1")
			roleManager1 := roleManager.WithTenant("tenant-1")
			role1, err := roleManager1.Registrar.Register(ctx, "admin", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(role1.TenantID).ToNot(BeNil())
			Expect(*role1.TenantID).To(Equal("tenant-1"))

			tenantPM.WithTenant("tenant-2")
			roleManager2 := roleManager.WithTenant("tenant-2")
			role2, err := roleManager2.Registrar.Register(ctx, "admin", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(role2.TenantID).ToNot(BeNil())
			Expect(*role2.TenantID).To(Equal("tenant-2"))

			Expect(role1.ID).ToNot(Equal(role2.ID))
		})
	})
})
