package config

import (
	"path/filepath"
	"testing"

	"github.com/glieske/forge/internal/platform"
)

func TestConfigSetGetAndSave(t *testing.T) {
	dir := t.TempDir()
	paths := platform.Paths{
		ConfigDir:  dir,
		ConfigPath: filepath.Join(dir, "config.toml"),
	}
	cfg := Default()
	if err := Set(&cfg, "repositories.channel", "beta"); err != nil {
		t.Fatal(err)
	}
	if err := Set(&cfg, "globals.services", "db,redis"); err != nil {
		t.Fatal(err)
	}
	if err := Save(paths.ConfigPath, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(paths)
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.Repositories.Channel; got != "beta" {
		t.Fatalf("channel = %q", got)
	}
	value, err := Get(loaded, "globals.services")
	if err != nil {
		t.Fatal(err)
	}
	if value != "db,redis" {
		t.Fatalf("globals.services = %q", value)
	}
}

func TestDefaultHasNoRepositoryURLs(t *testing.T) {
	cfg := Default()
	if cfg.Repositories.PluginsURL != "" {
		t.Fatalf("plugins url = %q", cfg.Repositories.PluginsURL)
	}
	if cfg.Repositories.UpdatesURL != "" {
		t.Fatalf("updates url = %q", cfg.Repositories.UpdatesURL)
	}
	missing := MissingRepositorySettings(cfg)
	if len(missing) != 2 {
		t.Fatalf("missing = %#v", missing)
	}
	if err := RequirePluginsURL(cfg); err == nil {
		t.Fatal("expected missing plugins URL error")
	}
	if err := RequireUpdatesURL(cfg); err == nil {
		t.Fatal("expected missing updates URL error")
	}
}

func TestDefaultAWSProfilesIsEmpty(t *testing.T) {
	cfg := Default()
	values, ok := cfg.Globals["aws_profiles"]
	if !ok {
		t.Fatal("aws_profiles key is missing")
	}
	if len(values) != 0 {
		t.Fatalf("aws_profiles = %#v", values)
	}
}
