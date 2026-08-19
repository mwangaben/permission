package tests

import (
	"context"
	"gorm.io/gorm"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mwangaben/permission/permission"
	"github.com/mwangaben/permission/role"
)

func TestGateTest(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gate tests")

	var _ = Describe("Guard", func() {
		var (
			db      *gorm.DB
			pm      *permission.Manager
			cleanup func()
			ctx     context.Context
		)

		BeforeEach(func() {
			db, cleanup = NewTestDBPost()
			pm = permission.NewManager(db)
			ctx = context.Background()

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

		AfterEach(func() {
			if cleanup != nil {
				cleanup()
			}
		})

		Describe("Authorization checks", func() {
			var guard *permission.Guard

			BeforeEach(func() {
				checker := permission.NewChecker(db, pm.Config, pm.Tenant)
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
}
