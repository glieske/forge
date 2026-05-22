package selfupdate

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/glieske/forge/internal/config"
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
