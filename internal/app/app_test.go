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
