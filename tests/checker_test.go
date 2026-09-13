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

var _ = Describe("Permission Checker", func() {
	var (
		ctx     context.Context
		db      *gorm.DB
		cleanup func()
		pm      *permission.Manager
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

	Context("When checking permissions via roles", func() {
		BeforeEach(func() {
			perm1, err := pm.Registrar.Register(ctx, "user.view", "web")
			Expect(err).ToNot(HaveOccurred())
			perm2, err := pm.Registrar.Register(ctx, "user.edit", "web")
			Expect(err).ToNot(HaveOccurred())

			roleManager := role.NewManager(pm.Repo, pm.Config, pm.Tenant)
			roleObj, err := roleManager.Registrar.Register(ctx, "editor", "web")
			Expect(err).ToNot(HaveOccurred())

			permManager := role.NewPermissionManager(pm.Repo)
			err = permManager.AssignPermissionToRole(ctx, perm1.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())
			err = permManager.AssignPermissionToRole(ctx, perm2.ID, roleObj.ID)
			Expect(err).ToNot(HaveOccurred())

			assigner := role.NewAssigner(pm.Repo)
			err = assigner.AssignRoleToModel(ctx, roleObj.ID, "user", 1, nil)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should have permission via role", func() {
			checker := permission.NewChecker(pm.Repo, pm.Config, pm.Tenant)
			has, err := checker.HasPermission(ctx, "user", 1, "user.view", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeTrue())
		})

		It("should have another permission via role", func() {
			checker := permission.NewChecker(pm.Repo, pm.Config, pm.Tenant)
			has, err := checker.HasPermission(ctx, "user", 1, "user.edit", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeTrue())
		})

		It("should not have permission not assigned", func() {
			checker := permission.NewChecker(pm.Repo, pm.Config, pm.Tenant)
			has, err := checker.HasPermission(ctx, "user", 1, "user.delete", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeFalse())
		})
	})

	Context("When assigning direct permissions", func() {
		BeforeEach(func() {
			perm, err := pm.Registrar.Register(ctx, "special.access", "web")
			Expect(err).ToNot(HaveOccurred())

			directAssigner := permission.NewDirectAssigner(pm.Repo)
			err = directAssigner.AssignPermissionToModel(ctx, perm.ID, "user", 2, nil)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should have direct permission", func() {
			checker := permission.NewChecker(pm.Repo, pm.Config, pm.Tenant)
			has, err := checker.HasPermission(ctx, "user", 2, "special.access", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeTrue())
		})

		It("should not have direct permission for another user", func() {
			checker := permission.NewChecker(pm.Repo, pm.Config, pm.Tenant)
			has, err := checker.HasPermission(ctx, "user", 3, "special.access", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(has).To(BeFalse())
		})
	})
})

// Keep this reference alive so the `storage` import is used even if
// future edits remove the only direct mention. Remove when no longer needed.
var _ storage.Repository
