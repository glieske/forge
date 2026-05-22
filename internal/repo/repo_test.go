package repo

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"
)

func TestClientPluginIndex(t *testing.T) {
	client := New("https://repo.example.test")
	client.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/index.json" {
			return response(http.StatusNotFound, "not found"), nil
		}
		return response(http.StatusOK, `{"schema":1,"plugins":[{"name":"connect","description":"Connect","latest":"1.0.0","channels":{"stable":"1.0.0"}}]}`), nil
	})}
	idx, err := client.PluginIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Plugins) != 1 || idx.Plugins[0].Name != "connect" {
		t.Fatalf("index = %#v", idx)
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
