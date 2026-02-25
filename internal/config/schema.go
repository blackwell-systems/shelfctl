package config

// Config is the top-level shelfctl configuration.
type Config struct {
	GitHub    GitHubConfig    `mapstructure:"github"`
	Defaults  DefaultsConfig  `mapstructure:"defaults"`
	Shelves   []ShelfConfig   `mapstructure:"shelves"`
	Migration MigrationConfig `mapstructure:"migration"`
}

// GitHubConfig holds GitHub API connection settings.
type GitHubConfig struct {
	Owner    string `mapstructure:"owner"`
	TokenEnv string `mapstructure:"token_env"`
	APIBase  string `mapstructure:"api_base"`
	Backend  string `mapstructure:"backend"`
	Token    string `mapstructure:"-"` // resolved at runtime, never written
}

// DefaultsConfig holds default values for operations.
type DefaultsConfig struct {
	Release     string `mapstructure:"release"`
	CacheDir    string `mapstructure:"cache_dir"`
	AssetNaming string `mapstructure:"asset_naming"` // "id" or "original"
}

// ShelfConfig defines a single shelf (topic-based document collection).
type ShelfConfig struct {
	Name           string `mapstructure:"name"`
	Owner          string `mapstructure:"owner"`
	Repo           string `mapstructure:"repo"`
	CatalogPath    string `mapstructure:"catalog_path"`
	DefaultRelease string `mapstructure:"default_release"`
}

// MigrationConfig holds settings for migrating files from other repos.
type MigrationConfig struct {
	Sources []MigrationSource `mapstructure:"sources"`
}

// MigrationSource defines a source repository for migration.
type MigrationSource struct {
	Owner   string            `mapstructure:"owner"`
	Repo    string            `mapstructure:"repo"`
	Ref     string            `mapstructure:"ref"`
	Mapping map[string]string `mapstructure:"mapping"`
}

// ShelfByName returns the shelf config with the given name, or nil.
func (c *Config) ShelfByName(name string) *ShelfConfig {
	for i := range c.Shelves {
		if c.Shelves[i].Name == name {
			return &c.Shelves[i]
		}
	}
	return nil
}

// EffectiveOwner returns the shelf's owner or falls back to the global owner.
func (s *ShelfConfig) EffectiveOwner(globalOwner string) string {
	if s.Owner != "" {
		return s.Owner
	}
	return globalOwner
}

// EffectiveRelease returns the shelf's default release or falls back to the global default.
func (s *ShelfConfig) EffectiveRelease(globalDefault string) string {
	if s.DefaultRelease != "" {
		return s.DefaultRelease
	}
	if globalDefault != "" {
		return globalDefault
	}
	return "library"
}

// EffectiveCatalogPath returns the catalog path for this shelf.
func (s *ShelfConfig) EffectiveCatalogPath() string {
	if s.CatalogPath != "" {
		return s.CatalogPath
	}
	return "catalog.yml"
}
