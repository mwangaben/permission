package tests

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"

	"github.com/mwangaben/permission/models"
	"github.com/mwangaben/permission/permission"
	"github.com/mwangaben/permission/role"
	"gorm.io/gorm"
)

func TestPermission(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Permission Suite")
}

var _ = Describe("Permission System", func() {
	var (
		db      *gorm.DB
		cleanup func()
		pm      *permission.Manager
	)

	BeforeEach(func() {
		db, cleanup = NewTestDB()
		pm = permission.NewManager(db)
	})

	AfterEach(func() {
		if cleanup != nil {
			cleanup()
		}
	})

	Context("Permission Registration", func() {
		It("should register a new permission", func() {
			perm, err := pm.Registrar.Register("user.view", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(perm.Name).To(Equal("user.view"))
			Expect(perm.GuardName).To(Equal("web"))
		})

		It("should find a permission by name", func() {
			_, err := pm.Registrar.Register("user.view", "web")
			Expect(err).ToNot(HaveOccurred())

			found, err := pm.Registrar.FindByName("user.view", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(found.Name).To(Equal("user.view"))
		})

		It("should register multiple permissions", func() {
			permissions := []struct{ Name, GuardName string }{
				{"user.view", "web"},
				{"user.create", "web"},
				{"user.update", "web"},
			}
			created, err := pm.Registrar.RegisterMany(permissions)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(created)).To(Equal(3))
		})

		It("should register global permission", func() {
			globalPerm, err := pm.Registrar.RegisterGlobal("system.view", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(globalPerm.TenantID).To(BeNil())
		})
	})

	Context("Role Registration", func() {
		It("should register a new role", func() {
			roleManager := role.NewManager(db, pm.Config, pm.Tenant)
			roleObj, err := roleManager.Registrar.Register("admin", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(roleObj.Name).To(Equal("admin"))
			Expect(roleObj.GuardName).To(Equal("web"))
		})

		It("should find a role by name", func() {
			roleManager := role.NewManager(db, pm.Config, pm.Tenant)
			_, err := roleManager.Registrar.Register("admin", "web")
			Expect(err).ToNot(HaveOccurred())

			found, err := roleManager.Registrar.FindByName("admin", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(found.Name).To(Equal("admin"))
		})

		It("should register tenant-specific role", func() {
			pm.EnableTenant("string")
			pm.WithTenant("tenant-1")

			roleManager := role.NewManager(db, pm.Config, pm.Tenant)
			roleObj, err := roleManager.Registrar.Register("tenant-admin", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(roleObj.TenantID).ToNot(BeNil())
			Expect(*roleObj.TenantID).To(Equal("tenant-1"))
		})
	})

	Context("Role Permission Assignment", func() {
		var (
			perm1       *models.Permission
			perm2       *models.Permission
			roleObj     *models.Role
			roleManager *role.Manager
			permManager *role.PermissionManager
		)

		BeforeEach(func() {
			var err error
			// Register permissions
			perm1, err = pm.Registrar.Register("user.view", "web")
			Expect(err).ToNot(HaveOccurred())
			perm2, err = pm.Registrar.Register("user.edit", "web")
			Expect(err).ToNot(HaveOccurred())

			// Create role
			roleManager = role.NewManager(db, pm.Config, pm.Tenant)
			roleObj, err = roleManager.Registrar.Register("editor", "web")
			Expect(err).ToNot(HaveOccurred())

			permManager = role.NewPermissionManager(db)
		})

		It("should assign permissions to a role", func() {
			err := permManager.AssignPermissionToRole(perm1.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			err = permManager.AssignPermissionToRole(perm2.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())

			// Get permissions for role
			perms, err := permManager.GetPermissionsForRole(roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(perms).To(HaveLen(2))

			// Reload role with permissions
			reloadedRole, err := roleManager.Registrar.FindByName(roleObj.Name, "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(reloadedRole.HasPermission("user.view")).To(BeTrue())
			Expect(reloadedRole.HasPermission("user.edit")).To(BeTrue())
		})

		It("should assign permission to role by name", func() {
			err := permManager.AssignPermissionToRoleByName("user.view", "editor", "web")
			Expect(err).ToNot(HaveOccurred())

			// Verify assignment
			perms, err := permManager.GetPermissionsForRole(roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(perms).To(HaveLen(1))
			Expect(perms[0].Name).To(Equal("user.view"))
		})

		It("should remove/revoke a permission from a role", func() {
			// First assign permissions
			err := permManager.AssignPermissionToRole(perm1.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			err = permManager.AssignPermissionToRole(perm2.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())

			// Verify both permissions are assigned
			perms, err := permManager.GetPermissionsForRole(roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(perms).To(HaveLen(2))

			// Remove one permission
			err = permManager.RemovePermissionFromRole(perm1.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())

			// Verify only one permission remains
			perms, err = permManager.GetPermissionsForRole(roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(perms).To(HaveLen(1))
			Expect(perms[0].Name).To(Equal("user.edit"))
		})

		It("should remove/revoke a permission from a role by name", func() {
			// First assign permissions
			err := permManager.AssignPermissionToRole(perm1.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			err = permManager.AssignPermissionToRole(perm2.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())

			// Verify both permissions are assigned
			perms, err := permManager.GetPermissionsForRole(roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(perms).To(HaveLen(2))

			// Remove one permission by name
			err = permManager.RemovePermissionFromRoleByName("user.view", "editor", "web")
			Expect(err).ToNot(HaveOccurred())

			// Verify only one permission remains
			perms, err = permManager.GetPermissionsForRole(roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(perms).To(HaveLen(1))
			Expect(perms[0].Name).To(Equal("user.edit"))
		})

		It("should check if role has a specific permission", func() {
			err := permManager.AssignPermissionToRole(perm1.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())

			has, err := permManager.HasPermissionForRole(roleObj.ID, "user.view", nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeTrue())

			has, err = permManager.HasPermissionForRole(roleObj.ID, "user.delete", nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeFalse())
		})

		It("should get all roles with a specific permission", func() {
			// Create another role
			roleManager := role.NewManager(db, pm.Config, pm.Tenant)
			role2, err := roleManager.Registrar.Register("viewer", "web")
			Expect(err).ToNot(HaveOccurred())

			// Assign same permission to both roles
			err = permManager.AssignPermissionToRole(perm1.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			err = permManager.AssignPermissionToRole(perm1.ID, role2.ID)
			Expect(err).ToNot(HaveOccurred())

			// Get roles with permission
			roles, err := permManager.GetRolesWithPermission("user.view", nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(roles).To(HaveLen(2))
		})

		It("should sync permissions for a role", func() {
			// Register a third permission
			perm3, err := pm.Registrar.Register("user.delete", "web")
			Expect(err).ToNot(HaveOccurred())

			// Assign initial permissions
			err = permManager.AssignPermissionToRole(perm1.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			err = permManager.AssignPermissionToRole(perm2.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())

			// Sync permissions (only keep perm2 and perm3)
			err = permManager.SyncPermissionsForRole(roleObj.ID, []uint{perm2.ID, perm3.ID})
			Expect(err).ToNot(HaveOccurred())

			// Verify only perm2 and perm3 remain
			perms, err := permManager.GetPermissionsForRole(roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(perms).To(HaveLen(2))
			Expect(perms[0].Name).To(Equal("user.edit"))
			Expect(perms[1].Name).To(Equal("user.delete"))
		})
	})

	Context("Direct Permission Assignment to Models", func() {
		var (
			perm           *models.Permission
			directAssigner *permission.DirectAssigner
		)

		BeforeEach(func() {
			var err error
			// Register permission
			perm, err = pm.Registrar.Register("special.access", "web")
			Expect(err).ToNot(HaveOccurred())

			directAssigner = permission.NewDirectAssigner(db)
		})

		It("should assign permission directly to a model", func() {
			err := directAssigner.AssignPermissionToModel(perm.ID, "user", 2)
			Expect(err).ToNot(HaveOccurred())

			// Verify direct permission
			checker := permission.NewChecker(db, pm.Config, pm.Tenant)
			has, err := checker.HasPermission("user", 2, "special.access", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeTrue())
		})

		It("should assign permission directly to a model by name", func() {
			err := directAssigner.AssignPermissionToModelByName("special.access", "user", 3, "web")
			Expect(err).ToNot(HaveOccurred())

			// Verify direct permission
			checker := permission.NewChecker(db, pm.Config, pm.Tenant)
			has, err := checker.HasPermission("user", 3, "special.access", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeTrue())
		})

		It("should remove direct permission from a model", func() {
			// First assign permission
			err := directAssigner.AssignPermissionToModel(perm.ID, "user", 4)
			Expect(err).ToNot(HaveOccurred())

			// Verify permission exists
			checker := permission.NewChecker(db, pm.Config, pm.Tenant)
			has, err := checker.HasPermission("user", 4, "special.access", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeTrue())

			// Remove permission
			err = directAssigner.RemovePermissionFromModel(perm.ID, "user", 4)
			Expect(err).ToNot(HaveOccurred())

			// Verify permission is removed
			has, err = checker.HasPermission("user", 4, "special.access", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeFalse())
		})

		It("should get direct permissions for a model", func() {
			// Register another permission
			perm2, err := pm.Registrar.Register("admin.access", "web")
			Expect(err).ToNot(HaveOccurred())

			// Assign multiple permissions
			err = directAssigner.AssignPermissionToModel(perm.ID, "user", 5)
			Expect(err).ToNot(HaveOccurred())
			err = directAssigner.AssignPermissionToModel(perm2.ID, "user", 5)
			Expect(err).ToNot(HaveOccurred())

			// Get direct permissions
			perms, err := directAssigner.GetDirectPermissionsForModel("user", 5)
			Expect(err).ToNot(HaveOccurred())
			Expect(perms).To(HaveLen(2))
		})

		It("should check if model has direct permission", func() {
			err := directAssigner.AssignPermissionToModel(perm.ID, "user", 6)
			Expect(err).ToNot(HaveOccurred())

			has, err := directAssigner.HasDirectPermission("user", 6, "special.access", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeTrue())

			has, err = directAssigner.HasDirectPermission("user", 6, "admin.access", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeFalse())
		})

		It("should sync direct permissions for a model", func() {
			// Register another permission
			perm2, err := pm.Registrar.Register("admin.access", "web")
			Expect(err).ToNot(HaveOccurred())
			perm3, err := pm.Registrar.Register("moderator.access", "web")
			Expect(err).ToNot(HaveOccurred())

			// Assign initial permissions
			err = directAssigner.AssignPermissionToModel(perm.ID, "user", 7)
			Expect(err).ToNot(HaveOccurred())
			err = directAssigner.AssignPermissionToModel(perm2.ID, "user", 7)
			Expect(err).ToNot(HaveOccurred())

			// Sync permissions (only keep perm2 and perm3)
			err = directAssigner.SyncPermissionsForModel("user", 7, []uint{perm2.ID, perm3.ID})
			Expect(err).ToNot(HaveOccurred())

			// Verify only perm2 and perm3 remain
			perms, err := directAssigner.GetDirectPermissionsForModel("user", 7)
			Expect(err).ToNot(HaveOccurred())
			Expect(perms).To(HaveLen(2))
			permNames := []string{perms[0].Name, perms[1].Name}
			Expect(permNames).To(ContainElement("admin.access"))
			Expect(permNames).To(ContainElement("moderator.access"))
		})
	})
})
