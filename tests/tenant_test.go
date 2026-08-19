package tests

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"

	"github.com/mwangaben/permission/config"
	"github.com/mwangaben/permission/permission"
	"github.com/mwangaben/permission/role"
	"gorm.io/gorm"
)

func TestTenant(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tenant assignment to Role and Permission")
}

var _ = Describe("Tenant Permissions", func() {
	var (
		db      *gorm.DB
		cleanup func()
		pm      *permission.Manager
	)

	BeforeEach(func() {
		db, cleanup = NewTestDBPost()
		pm = permission.NewManager(
			db,
			config.WithTenant("uint"),
		)
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
			perm, err := pm.Registrar.Register("tenant.data.view", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(perm.TenantID).ToNot(BeNil())
			Expect(*perm.TenantID).To(Equal("1"))
		})

		It("should register global permission without tenant_id", func() {
			globalPerm, err := pm.Registrar.RegisterGlobal("system.view", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(globalPerm.TenantID).To(BeNil())
		})

		It("should create role with tenant context", func() {
			roleManager := role.NewManager(db, pm.Config, pm.Tenant)
			roleObj, err := roleManager.Registrar.Register("tenant-admin", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(roleObj.TenantID).ToNot(BeNil())
			Expect(*roleObj.TenantID).To(Equal("1"))
		})
	})

	Context("When switching tenants", func() {
		It("should create tenant-specific permissions for different tenants", func() {
			// Register permission for tenant-1
			pm.WithTenant("1")
			perm1, err := pm.Registrar.Register("user.view", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(perm1.TenantID).ToNot(BeNil())
			Expect(*perm1.TenantID).To(Equal("1"))

			// Switch to tenant-2 and register same permission name
			pm.WithTenant("2")
			perm2, err := pm.Registrar.Register("user.view", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(perm2.TenantID).ToNot(BeNil())
			Expect(*perm2.TenantID).To(Equal("2"))

			// Verify they are different records
			Expect(perm1.ID).ToNot(Equal(perm2.ID))
		})

		It("should create tenant-specific roles for different tenants", func() {
			// Create role manager
			roleManager := role.NewManager(db, pm.Config, pm.Tenant)

			// Create role for tenant-1 using WithTenant on role manager
			roleManager1 := roleManager.WithTenant("1")
			role1, err := roleManager1.Registrar.Register("admin", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(role1.TenantID).ToNot(BeNil())
			Expect(*role1.TenantID).To(Equal("1"))

			// Create role for tenant-2 using WithTenant on role manager
			roleManager2 := roleManager.WithTenant("2")
			role2, err := roleManager2.Registrar.Register("admin", "web")
			Expect(err).ToNot(HaveOccurred())
			Expect(role2.TenantID).ToNot(BeNil())
			Expect(*role2.TenantID).To(Equal("2"))

			Expect(role1.ID).ToNot(Equal(role2.ID))
		})
	})
})
