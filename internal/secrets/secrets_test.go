package secrets

import (
	"path/filepath"
	"testing"
)

func TestFileStore(t *testing.T) {
	store := FileStore{Path: filepath.Join(t.TempDir(), "secrets.toml")}
	if err := store.Set("plugin:connect", "token", "secret"); err != nil {
		t.Fatal(err)
	}
	value, err := store.Get("plugin:connect", "token")
	if err != nil {
		t.Fatal(err)
	}
	if value != "secret" {
		t.Fatalf("secret = %q", value)
	}
	if err := store.Delete("plugin:connect", "token"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get("plugin:connect", "token"); err == nil {
		t.Fatal("expected missing secret")
	}
}
