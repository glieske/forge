package plugin

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/glieske/forge/internal/config"
	"github.com/glieske/forge/internal/platform"
	"github.com/glieske/forge/internal/repo"
	"github.com/glieske/forge/internal/security"
)

func TestInstallPluginFromHTTPRepo(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	packageFile := packageName("connect", "1.2.0", runtime.GOOS, runtime.GOARCH)
	archive := testTarGz(t, "forge-connect", "#!/usr/bin/env sh\nprintf ok\n")
	checksums := fmt.Sprintf("%s  %s\n", security.SHA256(archive), packageFile)
	sig := base64.StdEncoding.EncodeToString(ed25519.Sign(priv, []byte(checksums)))
	manifest := `
schema = 1
name = "connect"
version = "1.2.0"
description = "Connect"
entrypoint = "forge-connect"

[[commands]]
name = "connect"
description = "Connect"
`
	client := repo.New("https://repo.example.test")
	client.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/index.json":
			return response(http.StatusOK, `{"schema":1,"plugins":[{"name":"connect","description":"Connect","latest":"1.2.0","channels":{"stable":"1.2.0"}}]}`), nil
		case "/connect/index.json":
			return response(http.StatusOK, `{"schema":1,"name":"connect","versions":["1.2.0"],"channels":{"stable":"1.2.0"}}`), nil
		case "/connect/1.2.0/" + packageFile:
			return byteResponse(http.StatusOK, archive), nil
		case "/connect/1.2.0/checksums.txt":
			return response(http.StatusOK, checksums), nil
		case "/connect/1.2.0/checksums.txt.sig":
			return response(http.StatusOK, sig), nil
		case "/connect/1.2.0/manifest.toml":
			return response(http.StatusOK, manifest), nil
		default:
			return response(http.StatusNotFound, "not found"), nil
		}
	})}

	root := t.TempDir()
	mgr := Manager{
		Paths: platform.Paths{
			ConfigPath: filepath.Join(root, "config.toml"),
			DataDir:    filepath.Join(root, "data"),
			PluginDir:  filepath.Join(root, "data", "plugins"),
		},
		Config: config.Config{
			Repositories: config.RepositoriesConfig{Channel: "stable"},
			Security:     config.SecurityConfig{RequireChecksums: true, RequireSignatures: true, PublicKey: base64.StdEncoding.EncodeToString(pub)},
		},
		Repo: client,
	}
	if err := mgr.Install(t.Context(), "connect", "", "stable"); err != nil {
		t.Fatal(err)
	}
	items, err := mgr.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Name != "connect" || items[0].Version != "1.2.0" {
		t.Fatalf("installed = %#v", items)
	}
}

func TestInstallPluginInstallsDependencies(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	type artifact struct {
		packageFile string
		archive     []byte
		checksums   string
		signature   string
	}
	artifacts := map[string]artifact{
		"base": {
			packageFile: packageName("base", "1.0.0", runtime.GOOS, runtime.GOARCH),
			archive:     testTarGz(t, "forge-base", "#!/usr/bin/env sh\nprintf base\n"),
		},
		"connect": {
			packageFile: packageName("connect", "1.2.0", runtime.GOOS, runtime.GOARCH),
			archive:     testTarGz(t, "forge-connect", "#!/usr/bin/env sh\nprintf connect\n"),
		},
	}
	for name, item := range artifacts {
		item.checksums = fmt.Sprintf("%s  %s\n", security.SHA256(item.archive), item.packageFile)
		item.signature = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, []byte(item.checksums)))
		artifacts[name] = item
	}
	manifests := map[string]string{
		"/base/1.0.0/manifest.toml": `
schema = 1
name = "base"
version = "1.0.0"
description = "Base"
entrypoint = "forge-base"
`,
		"/connect/1.2.0/manifest.toml": `
schema = 1
name = "connect"
version = "1.2.0"
description = "Connect"
entrypoint = "forge-connect"

[[dependencies]]
name = "base"
version = ">=1.0.0 <2.0.0"
`,
	}
	client := repo.New("https://repo.example.test")
	client.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/connect/index.json":
			return response(http.StatusOK, `{"schema":1,"name":"connect","versions":["1.2.0"],"channels":{"stable":"1.2.0"}}`), nil
		case "/base/index.json":
			return response(http.StatusOK, `{"schema":1,"name":"base","versions":["1.0.0"],"channels":{"stable":"1.0.0"}}`), nil
		case "/connect/1.2.0/manifest.toml", "/base/1.0.0/manifest.toml":
			return response(http.StatusOK, manifests[r.URL.Path]), nil
		case "/connect/1.2.0/" + artifacts["connect"].packageFile:
			return byteResponse(http.StatusOK, artifacts["connect"].archive), nil
		case "/connect/1.2.0/checksums.txt":
			return response(http.StatusOK, artifacts["connect"].checksums), nil
		case "/connect/1.2.0/checksums.txt.sig":
			return response(http.StatusOK, artifacts["connect"].signature), nil
		case "/base/1.0.0/" + artifacts["base"].packageFile:
			return byteResponse(http.StatusOK, artifacts["base"].archive), nil
		case "/base/1.0.0/checksums.txt":
			return response(http.StatusOK, artifacts["base"].checksums), nil
		case "/base/1.0.0/checksums.txt.sig":
			return response(http.StatusOK, artifacts["base"].signature), nil
		default:
			return response(http.StatusNotFound, "not found"), nil
		}
	})}
	root := t.TempDir()
	mgr := Manager{
		Paths: platform.Paths{
			ConfigPath: filepath.Join(root, "config.toml"),
			DataDir:    filepath.Join(root, "data"),
			PluginDir:  filepath.Join(root, "data", "plugins"),
		},
		Config: config.Config{
			Repositories: config.RepositoriesConfig{Channel: "stable"},
			Security:     config.SecurityConfig{RequireChecksums: true, RequireSignatures: true, PublicKey: base64.StdEncoding.EncodeToString(pub)},
		},
		Repo: client,
	}
	if err := mgr.Install(t.Context(), "connect", "", "stable"); err != nil {
		t.Fatal(err)
	}
	items, err := mgr.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected connect and base installed, got %#v", items)
	}
}

func TestDescribeInstalledPlugin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixture is unix-only")
	}
	root := t.TempDir()
	current := filepath.Join(root, "data", "plugins", "echo", "versions", "1.0.0")
	if err := os.MkdirAll(current, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(current, "manifest.toml"), []byte(`
schema = 1
name = "echo"
version = "1.0.0"
description = "Echo"
entrypoint = "forge-echo"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(current, "forge-echo"), []byte(`#!/usr/bin/env sh
if [ "$1" = "--forge-describe" ]; then
  printf '{"schema":1,"name":"echo","version":"1.0.0","summary":"Echo","capabilities":["describe"],"commands":[]}'
  exit 0
fi
exit 2
`), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "data", "plugins", "echo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("versions", "1.0.0"), filepath.Join(root, "data", "plugins", "echo", "current")); err != nil {
		t.Fatal(err)
	}
	mgr := Manager{Paths: platform.Paths{PluginDir: filepath.Join(root, "data", "plugins")}}
	desc, err := mgr.Describe(t.Context(), "echo")
	if err != nil {
		t.Fatal(err)
	}
	if desc.Name != "echo" || len(desc.Capabilities) != 1 {
		t.Fatalf("desc = %#v", desc)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func response(status int, body string) *http.Response {
	return byteResponse(status, []byte(body))
}

func byteResponse(status int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     http.Header{},
	}
}

func testTarGz(t *testing.T, name, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	data := []byte(content)
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(data))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
