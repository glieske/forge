package tui

import (
	"strings"
	"testing"

	"github.com/glieske/forge/internal/config"
	"github.com/glieske/forge/internal/platform"
	"github.com/glieske/forge/internal/plugin"
	"github.com/glieske/forge/internal/repo"
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

func TestShowInstalledIncludesInstallOption(t *testing.T) {
	model := New(Options{Config: config.Default()})
	model.showInstalled()
	first := model.list.Items()[0].(item)
	if first.title != "Install from Available plugins" || first.action != "available" {
		t.Fatalf("first = %#v", first)
	}
}

func TestShowAvailablePluginItemsInstallOnEnter(t *testing.T) {
	cfg := config.Default()
	cfg.Repositories.PluginsURL = "https://example.com/plugins"
	model := New(Options{Config: cfg})
	model.available = []repo.PluginSummary{
		{Name: "connect", Latest: "1.2.0", Description: "Connect to services"},
	}
	model.showAvailable()
	first := model.list.Items()[0].(item)
	if first.action != "install" || !strings.Contains(first.desc, "enter: install") {
		t.Fatalf("first = %#v", first)
	}
}

func TestShowUpdatesWithoutUpdatesURLShowsConfigurationWarning(t *testing.T) {
	model := New(Options{Config: config.Default()})
	model.showUpdates()
	view := model.View()
	if !strings.Contains(view, "Missing updates repository") {
		t.Fatalf("expected missing updates repository warning, got:\n%s", view)
	}
}

func TestShowUpdatesIncludesCheckApplyAndVersionActions(t *testing.T) {
	cfg := config.Default()
	cfg.Repositories.UpdatesURL = "https://example.com/updates"
	model := New(Options{Config: cfg})
	model.showUpdates()
	if len(model.list.Items()) != 6 {
		t.Fatalf("items = %#v", model.list.Items())
	}
	actions := []string{"update-check", "update-apply", "update-versions"}
	for i, action := range actions {
		got := model.list.Items()[i].(item)
		if got.action != action {
			t.Fatalf("item %d = %#v", i, got)
		}
	}
}

func TestShowUpdateVersionsAppliesSelectedVersion(t *testing.T) {
	model := New(Options{Config: config.Default()})
	model.showUpdateVersions([]string{"0.2.0"})
	first := model.list.Items()[0].(item)
	if first.action != "update-apply-version" || first.value != "0.2.0" {
		t.Fatalf("first = %#v", first)
	}
}

func TestChangeUpdateChannelUpdatesConfig(t *testing.T) {
	cfg := config.Default()
	cfg.Repositories.UpdatesURL = "https://example.com/updates"
	paths := platform.Paths{ConfigPath: t.TempDir() + "/config.toml"}
	model := New(Options{Config: cfg, Paths: paths})
	updated, _ := model.changeUpdateChannel("beta")
	updatedModel := updated.(Model)
	if updatedModel.opts.Config.Repositories.Channel != "beta" {
		t.Fatalf("channel = %q", updatedModel.opts.Config.Repositories.Channel)
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
