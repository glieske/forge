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
