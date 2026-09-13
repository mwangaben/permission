package tests

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mwangaben/permission/permission"
	"github.com/mwangaben/permission/role"
	"github.com/mwangaben/permission/storage"
	"gorm.io/gorm"
)

var _ = Describe("Permission System", func() {
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

	Context("Permission Registration", func() {
		It("should register a new permission", func() {
			perm, err := pm.Registrar.Register(ctx, "user.view", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(perm.Name).To(Equal("user.view"))
			Expect(perm.GuardName).To(Equal("web"))
		})

		It("should find a permission by name", func() {
			_, err := pm.Registrar.Register(ctx, "user.view", "web")
			Expect(err).ToNot(HaveOccurred())

			found, err := pm.Registrar.FindByName(ctx, "user.view", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(found.Name).To(Equal("user.view"))
		})

		It("should register multiple permissions", func() {
			permissions := []struct{ Name, GuardName string }{
				{"user.view", "web"},
				{"user.create", "web"},
				{"user.update", "web"},
			}
			created, err := pm.Registrar.RegisterMany(ctx, permissions)
			Expect(err).ToNot(HaveOccurred())
			Expect(created).To(HaveLen(3))
		})

		It("should register global permission", func() {
			globalPerm, err := pm.Registrar.RegisterGlobal(ctx, "system.view", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(globalPerm.TenantID).To(BeNil())
		})
	})

	Context("Role Registration", func() {
		It("should register a new role", func() {
			roleManager := role.NewManager(pm.Repo, pm.Config, pm.Tenant)
			roleObj, err := roleManager.Registrar.Register(ctx, "admin", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(roleObj.Name).To(Equal("admin"))
			Expect(roleObj.GuardName).To(Equal("web"))
		})

		It("should find a role by name", func() {
			roleManager := role.NewManager(pm.Repo, pm.Config, pm.Tenant)
			_, err := roleManager.Registrar.Register(ctx, "admin", "web")
			Expect(err).ToNot(HaveOccurred())

			found, err := roleManager.Registrar.FindByName(ctx, "admin", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(found.Name).To(Equal("admin"))
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
			permManager *role.PermissionManager
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

			permManager = role.NewPermissionManager(pm.Repo)
		})

		It("should assign permissions to a role", func() {
			err := permManager.AssignPermissionToRole(ctx, perm1.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			err = permManager.AssignPermissionToRole(ctx, perm2.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())

			perms, err := permManager.GetPermissionsForRole(ctx, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(perms).To(HaveLen(2))
		})

		It("should assign permission to role by name", func() {
			err := permManager.AssignPermissionToRoleByName(ctx, "user.view", "editor", "web", nil)
			Expect(err).ToNot(HaveOccurred())

			perms, err := permManager.GetPermissionsForRole(ctx, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(perms).To(HaveLen(1))
			Expect(perms[0].Name).To(Equal("user.view"))
		})

		It("should remove/revoke a permission from a role", func() {
			err := permManager.AssignPermissionToRole(ctx, perm1.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			err = permManager.AssignPermissionToRole(ctx, perm2.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())

			perms, err := permManager.GetPermissionsForRole(ctx, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(perms).To(HaveLen(2))

			err = permManager.RemovePermissionFromRole(ctx, perm1.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())

			perms, err = permManager.GetPermissionsForRole(ctx, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(perms).To(HaveLen(1))
			Expect(perms[0].Name).To(Equal("user.edit"))
		})

		It("should remove/revoke a permission from a role by name", func() {
			err := permManager.AssignPermissionToRole(ctx, perm1.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			err = permManager.AssignPermissionToRole(ctx, perm2.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())

			perms, err := permManager.GetPermissionsForRole(ctx, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(perms).To(HaveLen(2))

			err = permManager.RemovePermissionFromRoleByName(ctx, "user.view", "editor", "web", nil)
			Expect(err).ToNot(HaveOccurred())

			perms, err = permManager.GetPermissionsForRole(ctx, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(perms).To(HaveLen(1))
			Expect(perms[0].Name).To(Equal("user.edit"))
		})

		It("should check if role has a specific permission", func() {
			err := permManager.AssignPermissionToRole(ctx, perm1.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())

			has, err := permManager.HasPermissionForRole(ctx, roleObj.ID, "user.view", "web", nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeTrue())

			has, err = permManager.HasPermissionForRole(ctx, roleObj.ID, "user.delete", "web", nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeFalse())
		})

		It("should sync permissions for a role", func() {
			perm3, err := pm.Registrar.Register(ctx, "user.delete", "web")
			Expect(err).ToNot(HaveOccurred())

			err = permManager.AssignPermissionToRole(ctx, perm1.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			err = permManager.AssignPermissionToRole(ctx, perm2.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())

			err = permManager.SyncPermissionsForRole(ctx, roleObj.ID, []uint{perm2.ID, perm3.ID})
			Expect(err).ToNot(HaveOccurred())

			perms, err := permManager.GetPermissionsForRole(ctx, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(perms).To(HaveLen(2))
		})
	})

	Context("Direct Permission Assignment to Models", func() {
		var (
			perm           *storage.Permission
			directAssigner *permission.DirectAssigner
		)

		BeforeEach(func() {
			var err error
			perm, err = pm.Registrar.Register(ctx, "special.access", "web")
			Expect(err).ToNot(HaveOccurred())

			directAssigner = permission.NewDirectAssigner(pm.Repo)
		})

		It("should assign permission directly to a model", func() {
			err := directAssigner.AssignPermissionToModel(ctx, perm.ID, "user", 2, nil)
			Expect(err).ToNot(HaveOccurred())

			checker := permission.NewChecker(pm.Repo, pm.Config, pm.Tenant)
			has, err := checker.HasPermission(ctx, "user", 2, "special.access", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeTrue())
		})

		It("should assign permission directly to a model by name", func() {
			err := directAssigner.AssignPermissionToModelByName(ctx, "special.access", "user", 3, "web", nil)
			Expect(err).ToNot(HaveOccurred())

			checker := permission.NewChecker(pm.Repo, pm.Config, pm.Tenant)
			has, err := checker.HasPermission(ctx, "user", 3, "special.access", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeTrue())
		})

		It("should remove direct permission from a model", func() {
			err := directAssigner.AssignPermissionToModel(ctx, perm.ID, "user", 4, nil)
			Expect(err).ToNot(HaveOccurred())

			checker := permission.NewChecker(pm.Repo, pm.Config, pm.Tenant)
			has, err := checker.HasPermission(ctx, "user", 4, "special.access", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeTrue())

			err = directAssigner.RemovePermissionFromModel(ctx, perm.ID, "user", 4)
			Expect(err).ToNot(HaveOccurred())

			has, err = checker.HasPermission(ctx, "user", 4, "special.access", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeFalse())
		})

		It("should get direct permissions for a model", func() {
			perm2, err := pm.Registrar.Register(ctx, "admin.access", "web")
			Expect(err).ToNot(HaveOccurred())

			err = directAssigner.AssignPermissionToModel(ctx, perm.ID, "user", 5, nil)
			Expect(err).ToNot(HaveOccurred())
			err = directAssigner.AssignPermissionToModel(ctx, perm2.ID, "user", 5, nil)
			Expect(err).ToNot(HaveOccurred())

			perms, err := directAssigner.GetDirectPermissionsForModel(ctx, "user", 5, "web", nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(perms).To(HaveLen(2))
		})

		It("should check if model has direct permission", func() {
			err := directAssigner.AssignPermissionToModel(ctx, perm.ID, "user", 6, nil)
			Expect(err).ToNot(HaveOccurred())

			has, err := directAssigner.HasDirectPermission(ctx, "user", 6, "special.access", "web", nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeTrue())

			has, err = directAssigner.HasDirectPermission(ctx, "user", 6, "admin.access", "web", nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeFalse())
		})
	})
})
