package tests

import (
	"github.com/mwangaben/permission/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"

	"github.com/mwangaben/permission/config"
	"github.com/mwangaben/permission/permission"
	"github.com/mwangaben/permission/role"
	"gorm.io/gorm"
)

func TestRole(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Role Tests")
}

var _ = Describe("Role Management", func() {
	var (
		db      *gorm.DB
		cleanup func()
		pm      *permission.PermManager
	)

	BeforeEach(func() {
		db, cleanup = NewTestDB()
		pm = permission.NewPermManager(db)
	})

	AfterEach(func() {
		if cleanup != nil {
			cleanup()
		}
	})

	Context("Role Registration", func() {
		BeforeEach(func() {
			// Register permissions first
			_, err := pm.Registrar.Register("user.view", "web")
			Expect(err).ToNot(HaveOccurred())
			_, err = pm.Registrar.Register("user.edit", "web")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should register a new role", func() {
			roleManager := role.NewManager(db, pm.Config, pm.Tenant)
			roleObj, err := roleManager.Registrar.Register("editor", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(roleObj.Name).To(Equal("editor"))
			Expect(roleObj.GuardName).To(Equal("web"))
		})

		It("should return existing role when registering duplicate", func() {
			roleManager := role.NewManager(db, pm.Config, pm.Tenant)
			_, err := roleManager.Registrar.Register("editor", "web")
			Expect(err).ToNot(HaveOccurred())

			roleObj, err := roleManager.Registrar.Register("editor", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(roleObj.Name).To(Equal("editor"))
		})

		It("should find a role by name", func() {
			roleManager := role.NewManager(db, pm.Config, pm.Tenant)
			_, err := roleManager.Registrar.Register("editor", "web")
			Expect(err).ToNot(HaveOccurred())

			found, err := roleManager.Registrar.FindByName("editor", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(found.Name).To(Equal("editor"))
		})

		It("should register tenant-specific role", func() {
			// Enable tenant mode
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
		})

		It("should assign permissions to a role", func() {
			permManager := role.NewPermissionManager(db)
			err := permManager.AssignPermissionToRole(perm1.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			err = permManager.AssignPermissionToRole(perm2.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())

			// Get permissions for role
			perms, err := permManager.GetPermissionsForRole(roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(perms).To(HaveLen(2))

			// Reload role with permissions
			reloadedRole, err := roleManager.Registrar.FindByName("editor", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(reloadedRole.HasPermission("user.view")).To(BeTrue())
			Expect(reloadedRole.HasPermission("user.edit")).To(BeTrue())
		})
	})

	Context("Role Assignment to Models", func() {
		var (
			roleObj  *models.Role
			assigner *role.Assigner
		)

		BeforeEach(func() {
			// Register permission
			perm1, err := pm.Registrar.Register("user.view", "web")
			Expect(err).ToNot(HaveOccurred())

			// Create role
			roleManager := role.NewManager(db, pm.Config, pm.Tenant)
			roleObj, err = roleManager.Registrar.Register("viewer", "web")
			Expect(err).ToNot(HaveOccurred())

			// Assign permission to role
			permManager := role.NewPermissionManager(db)
			err = permManager.AssignPermissionToRole(perm1.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())

			// Create assigner
			assigner = role.NewAssigner(db)
		})

		It("should assign role to model", func() {
			err := assigner.AssignRoleToModel(roleObj.ID, "user", 1)
			Expect(err).ToNot(HaveOccurred())

			// Verify assignment
			roles, err := assigner.GetRolesForModel("user", 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(roles).ToNot(BeEmpty())
		})

		It("should assign role to model by name", func() {
			err := assigner.AssignRoleToModelByName("viewer", "user", 2, "web")
			Expect(err).ToNot(HaveOccurred())

			roles, err := assigner.GetRolesForModel("user", 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(roles).ToNot(BeEmpty())
		})

		It("should remove role from model", func() {
			// First assign role
			err := assigner.AssignRoleToModel(roleObj.ID, "user", 1)
			Expect(err).ToNot(HaveOccurred())

			// Then remove it
			err = assigner.RemoveRoleFromModel(roleObj.ID, "user", 1)
			Expect(err).ToNot(HaveOccurred())

			roles, err := assigner.GetRolesForModel("user", 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(roles).To(BeEmpty())
		})

		It("should sync roles for model", func() {
			roleManager := role.NewManager(db, pm.Config, pm.Tenant)
			role2, err := roleManager.Registrar.Register("editor", "web")
			Expect(err).ToNot(HaveOccurred())

			// Assign both roles to user
			err = assigner.AssignRoleToModel(roleObj.ID, "user", 3)
			Expect(err).ToNot(HaveOccurred())
			err = assigner.AssignRoleToModel(role2.ID, "user", 3)
			Expect(err).ToNot(HaveOccurred())

			// Sync roles (only keep role2)
			err = assigner.SyncRolesForModel("user", 3, []uint{role2.ID})
			Expect(err).ToNot(HaveOccurred())

			roles, err := assigner.GetRolesForModel("user", 3)
			Expect(err).ToNot(HaveOccurred())
			Expect(roles).To(HaveLen(1))
			Expect(roles[0].ID).To(Equal(role2.ID))
		})
	})

	Context("Role with Tenant Support", func() {
		var (
			tenantPM *permission.PermManager
		)

		BeforeEach(func() {
			// Create manager with tenant enabled
			tenantPM = permission.NewPermManager(
				db,
				config.WithTenant("string"),
			)
		})

		It("should create tenant-specific roles and permissions", func() {
			// Register permission for tenant-1
			tenantPM.WithTenant("tenant-1")
			perm1, err := tenantPM.Registrar.Register("user.view", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(perm1.TenantID).ToNot(BeNil())
			Expect(*perm1.TenantID).To(Equal("tenant-1"))

			// Create role for tenant-1
			roleManager := role.NewManager(db, tenantPM.Config, tenantPM.Tenant)
			role1, err := roleManager.Registrar.Register("admin", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(role1.TenantID).ToNot(BeNil())
			Expect(*role1.TenantID).To(Equal("tenant-1"))

			// Assign permission to role
			permManager := role.NewPermissionManager(db)
			err = permManager.AssignPermissionToRole(perm1.ID, role1.ID)
			Expect(err).ToNot(HaveOccurred())

			// Assign role to user (tenant-1)
			assigner := role.NewAssigner(db)
			err = assigner.AssignRoleToModel(role1.ID, "user", 1)
			Expect(err).ToNot(HaveOccurred())

			// Check permission for tenant-1
			checker := permission.NewChecker(db, tenantPM.Config, tenantPM.Tenant)
			has, err := checker.HasPermission("user", 1, "user.view", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeTrue())

			// Switch to tenant-2
			tenantPM.WithTenant("tenant-2")
			checker = permission.NewChecker(db, tenantPM.Config, tenantPM.Tenant)

			// Check permission for tenant-2 (should not have permission)
			has, err = checker.HasPermission("user", 1, "user.view", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeFalse())
		})

		It("should create tenant-specific roles for different tenants", func() {
			roleManager := role.NewManager(db, tenantPM.Config, tenantPM.Tenant)

			// Create role for tenant-1
			tenantPM.WithTenant("tenant-1")
			roleManager1 := roleManager.WithTenant("tenant-1")
			role1, err := roleManager1.Registrar.Register("admin", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(role1.TenantID).ToNot(BeNil())
			Expect(*role1.TenantID).To(Equal("tenant-1"))

			// Create role for tenant-2
			tenantPM.WithTenant("tenant-2")
			roleManager2 := roleManager.WithTenant("tenant-2")
			role2, err := roleManager2.Registrar.Register("admin", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(role2.TenantID).ToNot(BeNil())
			Expect(*role2.TenantID).To(Equal("tenant-2"))

			Expect(role1.ID).ToNot(Equal(role2.ID))
		})
	})
})
