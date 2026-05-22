package app

import (
	"bytes"
	"testing"
)

func TestConfigCommandUsesForgeHome(t *testing.T) {
	t.Setenv("FORGE_HOME", t.TempDir())
	cmd, err := NewRootCommand("0.1.0", "test")
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"config", "get", "repositories.channel"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if out.String() != "stable\n" {
		t.Fatalf("output = %q", out.String())
	}
}

func TestPluginAvailableRequiresPluginsURL(t *testing.T) {
	t.Setenv("FORGE_HOME", t.TempDir())
	cmd, err := NewRootCommand("0.1.0", "test")
	if err != nil {
		t.Fatal(err)
	}
	cmd.SetArgs([]string{"plugin", "available"})
	err = cmd.Execute()
	if err == nil {
		t.Fatal("expected missing plugins_url error")
	}
	if err.Error() != "missing repositories.plugins_url; set it with `forge config set repositories.plugins_url <url>`" {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestSelfUpdateCheckRequiresUpdatesURL(t *testing.T) {
	t.Setenv("FORGE_HOME", t.TempDir())
	cmd, err := NewRootCommand("0.1.0", "test")
	if err != nil {
		t.Fatal(err)
	}
	cmd.SetArgs([]string{"self-update", "check"})
	err = cmd.Execute()
	if err == nil {
		t.Fatal("expected missing updates_url error")
	}
	if err.Error() != "missing repositories.updates_url; set it with `forge config set repositories.updates_url <url>`" {
		t.Fatalf("error = %q", err.Error())
	}
}
