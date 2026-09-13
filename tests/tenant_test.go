package tests

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mwangaben/permission/config"
	"github.com/mwangaben/permission/permission"
	"github.com/mwangaben/permission/role"
	"gorm.io/gorm"
)

var _ = Describe("Tenant Permissions", func() {
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
		pm, err = permission.NewManager(db, config.WithTenant("uint"))
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if cleanup != nil {
			cleanup()
		}
	})

	Context("When tenant mode is enabled", func() {
		BeforeEach(func() {
			pm.WithTenant("1")
		})

		It("should register tenant-specific permission with tenant_id", func() {
			perm, err := pm.Registrar.Register(ctx, "tenant.data.view", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(perm.TenantID).ToNot(BeNil())
			Expect(*perm.TenantID).To(Equal("1"))
		})

		It("should register global permission without tenant_id", func() {
			globalPerm, err := pm.Registrar.RegisterGlobal(ctx, "system.view", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(globalPerm.TenantID).To(BeNil())
		})

		It("should create role with tenant context", func() {
			roleManager := role.NewManager(pm.Repo, pm.Config, pm.Tenant)
			roleObj, err := roleManager.Registrar.Register(ctx, "tenant-admin", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(roleObj.TenantID).ToNot(BeNil())
			Expect(*roleObj.TenantID).To(Equal("1"))
		})
	})

	Context("When switching tenants", func() {
		It("should create tenant-specific permissions for different tenants", func() {
			pm.WithTenant("1")
			perm1, err := pm.Registrar.Register(ctx, "user.view", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(perm1.TenantID).ToNot(BeNil())
			Expect(*perm1.TenantID).To(Equal("1"))

			pm.WithTenant("2")
			perm2, err := pm.Registrar.Register(ctx, "user.view", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(perm2.TenantID).ToNot(BeNil())
			Expect(*perm2.TenantID).To(Equal("2"))

			Expect(perm1.ID).ToNot(Equal(perm2.ID))
		})

		It("should create tenant-specific roles for different tenants", func() {
			roleManager := role.NewManager(pm.Repo, pm.Config, pm.Tenant)

			roleManager1 := roleManager.WithTenant("1")
			role1, err := roleManager1.Registrar.Register(ctx, "admin", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(role1.TenantID).ToNot(BeNil())
			Expect(*role1.TenantID).To(Equal("1"))

			roleManager2 := roleManager.WithTenant("2")
			role2, err := roleManager2.Registrar.Register(ctx, "admin", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(role2.TenantID).ToNot(BeNil())
			Expect(*role2.TenantID).To(Equal("2"))

			Expect(role1.ID).ToNot(Equal(role2.ID))
		})
	})
})
