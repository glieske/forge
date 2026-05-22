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
