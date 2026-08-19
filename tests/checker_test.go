package tests

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"

	"github.com/mwangaben/permission/permission"
	"github.com/mwangaben/permission/role"
	"gorm.io/gorm"
)

func TestChecker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Permission Checker Suite")
}

var _ = Describe("Permission Checker", func() {
	var (
		db      *gorm.DB
		cleanup func()
		pm      *permission.Manager
	)

	BeforeEach(func() {
		db, cleanup = NewTestDBPost()
		pm = permission.NewManager(db)
	})

	AfterEach(func() {
		if cleanup != nil {
			cleanup()
		}
	})

	Context("When checking permissions via roles", func() {
		BeforeEach(func() {
			// Register permissions
			perm1, err := pm.Registrar.Register("user.view", "web")
			Expect(err).ToNot(HaveOccurred())
			perm2, err := pm.Registrar.Register("user.edit", "web")
			Expect(err).ToNot(HaveOccurred())

			// Create role
			roleManager := role.NewManager(db, pm.Config, pm.Tenant)
			roleObj, err := roleManager.Registrar.Register("editor", "web")
			Expect(err).ToNot(HaveOccurred())

			// Assign permissions to role
			permManager := role.NewPermissionManager(db)
			err = permManager.AssignPermissionToRole(perm1.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			err = permManager.AssignPermissionToRole(perm2.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())

			// Assign role to user
			assigner := role.NewAssigner(db)
			err = assigner.AssignRoleToModel(roleObj.ID, "user", 1)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should have permission via role", func() {
			checker := permission.NewChecker(db, pm.Config, pm.Tenant)
			has, err := checker.HasPermission("user", 1, "user.view", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeTrue())
		})

		It("should have another permission via role", func() {
			checker := permission.NewChecker(db, pm.Config, pm.Tenant)
			has, err := checker.HasPermission("user", 1, "user.edit", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeTrue())
		})

		It("should not have permission not assigned", func() {
			checker := permission.NewChecker(db, pm.Config, pm.Tenant)
			has, err := checker.HasPermission("user", 1, "user.delete", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeFalse())
		})
	})

	Context("When assigning direct permissions", func() {
		BeforeEach(func() {
			// Register permission
			perm, err := pm.Registrar.Register("special.access", "web")
			Expect(err).ToNot(HaveOccurred())

			// Assign directly to user
			directAssigner := permission.NewDirectAssigner(db)
			err = directAssigner.AssignPermissionToModel(perm.ID, "user", 2)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should have direct permission", func() {
			checker := permission.NewChecker(db, pm.Config, pm.Tenant)
			has, err := checker.HasPermission("user", 2, "special.access", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeTrue())
		})

		It("should not have direct permission for another user", func() {
			checker := permission.NewChecker(db, pm.Config, pm.Tenant)
			has, err := checker.HasPermission("user", 3, "special.access", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeFalse())
		})
	})
})
