package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/glieske/forge/internal/platform"
)

const (
	DefaultPluginsURL = "https://example-bucket.s3.amazonaws.com/forge/plugins"
	DefaultUpdatesURL = "https://example-bucket.s3.amazonaws.com/forge/updates"
)

type Config struct {
	Repositories RepositoriesConfig     `toml:"repositories"`
	UI           UIConfig               `toml:"ui"`
	Security     SecurityConfig         `toml:"security"`
	Globals      map[string][]string    `toml:"globals"`
	Plugins      map[string]interface{} `toml:"plugins"`
}

type RepositoriesConfig struct {
	PluginsURL string `toml:"plugins_url"`
	UpdatesURL string `toml:"updates_url"`
	Channel    string `toml:"channel"`
}

type UIConfig struct {
	Interactive bool `toml:"interactive"`
	FuzzyLimit  int  `toml:"fuzzy_limit"`
}

type SecurityConfig struct {
	RequireChecksums  bool   `toml:"require_checksums"`
	RequireSignatures bool   `toml:"require_signatures"`
	PublicKey         string `toml:"public_key"`
	SecretsBackend    string `toml:"secrets_backend"`
}

func Default() Config {
	return Config{
		Repositories: RepositoriesConfig{PluginsURL: DefaultPluginsURL, UpdatesURL: DefaultUpdatesURL, Channel: "stable"},
		UI:           UIConfig{Interactive: true, FuzzyLimit: 50},
		Security:     SecurityConfig{RequireChecksums: true, RequireSignatures: true, SecretsBackend: "auto"},
		Globals: map[string][]string{
			"aws_profiles": {"dev", "stage", "prod"},
			"environments": {"dev", "stage", "prod"},
			"regions":      {"eu-central-1", "us-east-1"},
		},
		Plugins: map[string]interface{}{},
	}
}

func Load(paths platform.Paths) (Config, error) {
	cfg := Default()
	if _, err := os.Stat(paths.ConfigPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return Config{}, err
	}
	if _, err := toml.DecodeFile(paths.ConfigPath, &cfg); err != nil {
		return Config{}, err
	}
	applyEnv(&cfg)
	return cfg, nil
}

func EnsureFile(paths platform.Paths) error {
	if err := platform.Ensure(paths); err != nil {
		return err
	}
	if _, err := os.Stat(paths.ConfigPath); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return Save(paths.ConfigPath, Default())
}

func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}

func Set(cfg *Config, key, value string) error {
	switch key {
	case "repositories.plugins_url":
		cfg.Repositories.PluginsURL = value
	case "repositories.updates_url":
		cfg.Repositories.UpdatesURL = value
	case "repositories.channel":
		cfg.Repositories.Channel = value
	case "ui.interactive":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		cfg.UI.Interactive = v
	case "ui.fuzzy_limit":
		v, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		cfg.UI.FuzzyLimit = v
	case "security.public_key":
		cfg.Security.PublicKey = value
	case "security.secrets_backend":
		cfg.Security.SecretsBackend = value
	default:
		if strings.HasPrefix(key, "globals.") {
			if cfg.Globals == nil {
				cfg.Globals = map[string][]string{}
			}
			cfg.Globals[strings.TrimPrefix(key, "globals.")] = splitCSV(value)
			return nil
		}
		return fmt.Errorf("unsupported config key %q", key)
	}
	return nil
}

func Get(cfg Config, key string) (string, error) {
	switch key {
	case "repositories.plugins_url":
		return cfg.Repositories.PluginsURL, nil
	case "repositories.updates_url":
		return cfg.Repositories.UpdatesURL, nil
	case "repositories.channel":
		return cfg.Repositories.Channel, nil
	case "ui.interactive":
		return strconv.FormatBool(cfg.UI.Interactive), nil
	case "ui.fuzzy_limit":
		return strconv.Itoa(cfg.UI.FuzzyLimit), nil
	case "security.public_key":
		return cfg.Security.PublicKey, nil
	case "security.secrets_backend":
		return cfg.Security.SecretsBackend, nil
	default:
		if strings.HasPrefix(key, "globals.") {
			v := cfg.Globals[strings.TrimPrefix(key, "globals.")]
			return strings.Join(v, ","), nil
		}
		return "", fmt.Errorf("unsupported config key %q", key)
	}
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("FORGE_PLUGINS_URL"); v != "" {
		cfg.Repositories.PluginsURL = v
	}
	if v := os.Getenv("FORGE_UPDATES_URL"); v != "" {
		cfg.Repositories.UpdatesURL = v
	}
	if v := os.Getenv("FORGE_CHANNEL"); v != "" {
		cfg.Repositories.Channel = v
	}
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
