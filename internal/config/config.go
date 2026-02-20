package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// DefaultPath returns the default config file path.
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "shelfctl", "config.yml")
}

// Load reads the config from disk (or env). Returns an empty config if no
// file exists yet — init command will populate it.
func Load() (*Config, error) {
	v := viper.New()

	v.SetDefault("github.api_base", "https://api.github.com")
	v.SetDefault("github.token_env", "GITHUB_TOKEN")
	v.SetDefault("github.backend", "api")
	v.SetDefault("defaults.release", "library")
	v.SetDefault("defaults.asset_naming", "id")
	v.SetDefault("defaults.cache_dir", defaultCacheDir())
	v.SetDefault("serve.port", 8080)
	v.SetDefault("serve.host", "127.0.0.1")

	v.SetEnvPrefix("SHELFCTL")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	configPath := os.Getenv("SHELFCTL_CONFIG")
	if configPath == "" {
		configPath = DefaultPath()
	}
	v.SetConfigFile(configPath)

	if err := v.ReadInConfig(); err != nil {
		// Not finding the config file is fine — the init command creates it.
		if !os.IsNotExist(err) {
			if _, isCfgNotFound := err.(viper.ConfigFileNotFoundError); !isCfgNotFound {
				return nil, fmt.Errorf("reading config: %w", err)
			}
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Resolve token from env (never stored in file).
	tokenEnv := cfg.GitHub.TokenEnv
	if tokenEnv == "" {
		tokenEnv = "GITHUB_TOKEN"
	}
	cfg.GitHub.Token = os.Getenv(tokenEnv)
	if cfg.GitHub.Token == "" {
		cfg.GitHub.Token = os.Getenv("SHELFCTL_GITHUB_TOKEN")
	}

	// Expand ~ in cache dir.
	cfg.Defaults.CacheDir = ExpandHome(cfg.Defaults.CacheDir)

	return &cfg, nil
}

// Save writes the config to the default path.
func Save(cfg *Config) error {
	path := DefaultPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	return enc.Encode(cfg)
}

// ExpandHome expands a leading ~/ in a path.
func ExpandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func defaultCacheDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "shelfctl", "cache")
}
