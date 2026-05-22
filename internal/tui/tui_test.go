package tui

import (
	"testing"

	"github.com/glieske/forge/internal/config"
	"github.com/glieske/forge/internal/plugin"
)

func TestNewDashboardDoesNotRequireRepositoryData(t *testing.T) {
	model := New(Options{Version: "test", Config: config.Default()})
	model.showDashboard()
	view := model.View()
	if view == "" {
		t.Fatal("expected non-empty view")
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
