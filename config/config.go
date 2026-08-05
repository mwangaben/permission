package config

// Config holds the permission package configuration
type Config struct {
	// EnableTenant enables multi-tenant support
	// When enabled, all models will have tenant_id and queries will be scoped
	EnableTenant bool

	// TenantIDType specifies the type of tenant ID (string, uint, etc.)
	TenantIDType string // "string", "uint", "uuid"

	// DefaultGuard is the default guard name
	DefaultGuard string

	// TablePrefix allows custom table prefix
	TablePrefix string
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		EnableTenant: false,
		TenantIDType: "string",
		DefaultGuard: "web",
		TablePrefix:  "",
	}
}

// NewConfig creates a new configuration with options
func NewConfig(opts ...func(*Config)) *Config {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// WithTenant enables tenant support
func WithTenant(tenantIDType string) func(*Config) {
	return func(c *Config) {
		c.EnableTenant = true
		c.TenantIDType = tenantIDType
	}
}

// WithGuard sets the default guard
func WithGuard(guard string) func(*Config) {
	return func(c *Config) {
		c.DefaultGuard = guard
	}
}

// WithTablePrefix sets the table prefix
func WithTablePrefix(prefix string) func(*Config) {
	return func(c *Config) {
		c.TablePrefix = prefix
	}
}
