package tests

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mwangaben/permission/permission"
	"github.com/mwangaben/permission/role"
	"gorm.io/gorm"
)

var _ = Describe("Guard", func() {
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

	AfterEach(func() {
		if cleanup != nil {
			cleanup()
		}
	})

	Describe("Authorization checks", func() {
		var guard *permission.Guard

		BeforeEach(func() {
			checker := permission.NewChecker(pm.Repo, pm.Config, pm.Tenant)
			guard = permission.NewGuard(checker)
		})

		It("should allow with permission", func() {
			err := guard.Authorize(ctx, "user", 1, "user.view", "web")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should allow with another permission", func() {
			err := guard.Authorize(ctx, "user", 1, "user.edit", "web")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should deny without permission", func() {
			err := guard.Authorize(ctx, "user", 1, "user.delete", "web")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unauthorized"))
		})

		It("should allow any permission", func() {
			err := guard.AuthorizeAny(ctx, "user", 1, "web", "user.view", "user.delete")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should allow all permissions", func() {
			err := guard.AuthorizeAll(ctx, "user", 1, "web", "user.view", "user.edit")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should deny all permissions if missing one", func() {
			err := guard.AuthorizeAll(ctx, "user", 1, "web", "user.view", "user.delete")
			Expect(err).To(HaveOccurred())
		})
	})
})
