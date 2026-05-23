package selfupdate

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/glieske/forge/internal/config"
	"github.com/glieske/forge/internal/platform"
	"github.com/glieske/forge/internal/repo"
)

func TestCheckDetectsUpdate(t *testing.T) {
	client := repo.New("https://updates.example.test")
	client.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/stable/index.json" {
			return response(http.StatusNotFound, "not found"), nil
		}
		return response(http.StatusOK, `{"schema":1,"channel":"stable","latest":"0.2.0","minimum_supported":"0.1.0","versions":["0.2.0"]}`), nil
	})}
	mgr := Manager{
		Config:  config.Config{Repositories: config.RepositoriesConfig{Channel: "stable"}},
		Repo:    client,
		Version: "0.1.0",
	}
	res, err := mgr.Check(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !res.Update || res.Latest != "0.2.0" {
		t.Fatalf("result = %#v", res)
	}
}

func TestCheckVerifiesSignedUpdateIndex(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	index := `{"schema":1,"channel":"stable","latest":"0.2.0","minimum_supported":"0.1.0","versions":["0.2.0"]}`
	client := repo.New("https://updates.example.test")
	client.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/public-key.ed25519":
			return response(http.StatusOK, base64.StdEncoding.EncodeToString(pub)), nil
		case "/stable/index.json":
			return response(http.StatusOK, index), nil
		case "/stable/index.json.sig":
			sig := ed25519.Sign(priv, []byte(index))
			return response(http.StatusOK, base64.StdEncoding.EncodeToString(sig)), nil
		default:
			return response(http.StatusNotFound, "not found"), nil
		}
	})}
	root := t.TempDir()
	mgr := Manager{
		Config: config.Config{
			Repositories: config.RepositoriesConfig{Channel: "stable"},
			Security:     config.SecurityConfig{RequireSignatures: true},
		},
		Paths:   platform.Paths{ConfigDir: filepath.Join(root, "config")},
		Repo:    client,
		Version: "0.1.0",
	}
	res, err := mgr.Check(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !res.Update {
		t.Fatalf("result = %#v", res)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{},
	}
}
