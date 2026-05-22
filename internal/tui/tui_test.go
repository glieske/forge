package tui

import (
	"strings"
	"testing"

	"github.com/glieske/forge/internal/config"
	"github.com/glieske/forge/internal/platform"
	"github.com/glieske/forge/internal/plugin"
)

func TestNewDashboardDoesNotRequireRepositoryData(t *testing.T) {
	model := New(Options{Version: "test", Config: config.Default()})
	view := model.View()
	if !strings.Contains(view, "Dashboard") {
		t.Fatalf("expected dashboard in initial view, got:\n%s", view)
	}
	if !strings.Contains(view, "Installed plugins") {
		t.Fatalf("expected dashboard actions in initial view, got:\n%s", view)
	}
	if !strings.Contains(view, "Missing repositories.plugins_url") {
		t.Fatalf("expected missing plugins_url warning in initial view, got:\n%s", view)
	}
	if !strings.Contains(view, "Missing repositories.updates_url") {
		t.Fatalf("expected missing updates_url warning in initial view, got:\n%s", view)
	}
}

func TestShowAvailableWithoutPluginsURLShowsConfigurationWarning(t *testing.T) {
	model := New(Options{Config: config.Default()})
	model.showAvailable()
	view := model.View()
	if !strings.Contains(view, "Missing plugins repository") {
		t.Fatalf("expected missing repository warning, got:\n%s", view)
	}
}

func TestValuesForGlobalProvider(t *testing.T) {
	model := New(Options{Config: config.Default()})
	values := model.valuesFor(plugin.ArgSpec{
		ValueProvider:  "global.environments",
		FallbackValues: []string{"fallback"},
	})
	if len(values) != 3 || values[1] != "stage" {
		t.Fatalf("values = %#v", values)
	}
}

func TestValuesForStaticProvider(t *testing.T) {
	model := New(Options{Config: config.Default()})
	values := model.valuesFor(plugin.ArgSpec{ValueProvider: "static:db,redis"})
	if len(values) != 2 || values[0] != "db" || values[1] != "redis" {
		t.Fatalf("values = %#v", values)
	}
}

func TestShowConfigStartsWithSettingsMenu(t *testing.T) {
	model := New(Options{Config: config.Default()})
	model.showConfig()
	if len(model.list.Items()) != 2 {
		t.Fatalf("items = %#v", model.list.Items())
	}
}

func TestShowGlobalConfigIncludesConfigPath(t *testing.T) {
	model := New(Options{
		Config: config.Default(),
		Paths:  platform.Paths{ConfigPath: "/tmp/forge/config.toml"},
	})
	model.showGlobalConfig()
	first := model.list.Items()[0].(item)
	if first.title != "config file" || first.desc != "/tmp/forge/config.toml" {
		t.Fatalf("first = %#v", first)
	}
}

func TestShowPluginConfigUsesManifestFields(t *testing.T) {
	cfg := config.Default()
	config.SetPlugin(&cfg, "connect", "default_profile", "stage")
	model := New(Options{Config: cfg})
	model.showPluginConfig(plugin.Installed{
		Name:    "connect",
		Version: "1.2.0",
		Manifest: plugin.Manifest{
			Config: []plugin.ConfigSpec{
				{Key: "default_profile", Type: "string", Description: "Default profile"},
				{Key: "api_token", Type: "secret_string", Secret: true},
			},
		},
	})
	if len(model.list.Items()) != 2 {
		t.Fatalf("items = %#v", model.list.Items())
	}
	first := model.list.Items()[0].(item)
	if first.title != "default_profile" || !strings.Contains(first.desc, "stage") {
		t.Fatalf("first = %#v", first)
	}
	second := model.list.Items()[1].(item)
	if !strings.Contains(second.desc, "secret stored outside config") {
		t.Fatalf("second = %#v", second)
	}
}

func TestValidateConfigValue(t *testing.T) {
	if err := validateConfigValue(plugin.ConfigSpec{Key: "enabled", Type: "bool"}, "true"); err != nil {
		t.Fatal(err)
	}
	if err := validateConfigValue(plugin.ConfigSpec{Key: "enabled", Type: "bool"}, "not-bool"); err == nil {
		t.Fatal("expected bool validation error")
	}
	if err := validateConfigValue(plugin.ConfigSpec{Key: "port", Type: "int"}, "5432"); err != nil {
		t.Fatal(err)
	}
	if err := validateConfigValue(plugin.ConfigSpec{Key: "port", Type: "int"}, "abc"); err == nil {
		t.Fatal("expected int validation error")
	}
}
