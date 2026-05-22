package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type Client struct {
	BaseURL string
	HTTP    *http.Client
}

type PluginIndex struct {
	Schema      int             `json:"schema"`
	GeneratedAt string          `json:"generated_at"`
	Plugins     []PluginSummary `json:"plugins"`
}

type PluginSummary struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Latest      string            `json:"latest"`
	Channels    map[string]string `json:"channels"`
}

type PluginVersions struct {
	Schema   int               `json:"schema"`
	Name     string            `json:"name"`
	Versions []string          `json:"versions"`
	Channels map[string]string `json:"channels"`
}

type UpdateIndex struct {
	Schema           int      `json:"schema"`
	Channel          string   `json:"channel"`
	Latest           string   `json:"latest"`
	MinimumSupported string   `json:"minimum_supported"`
	Versions         []string `json:"versions"`
}

func New(baseURL string) Client {
	return Client{BaseURL: strings.TrimRight(baseURL, "/"), HTTP: &http.Client{Timeout: 30 * time.Second}}
}

func (c Client) PluginIndex(ctx context.Context) (PluginIndex, error) {
	var idx PluginIndex
	return idx, c.GetJSON(ctx, "index.json", &idx)
}

func (c Client) PluginVersions(ctx context.Context, name string) (PluginVersions, error) {
	var idx PluginVersions
	return idx, c.GetJSON(ctx, path.Join(name, "index.json"), &idx)
}

func (c Client) UpdateIndex(ctx context.Context, channel string) (UpdateIndex, error) {
	var idx UpdateIndex
	return idx, c.GetJSON(ctx, path.Join(channel, "index.json"), &idx)
}

func (c Client) GetJSON(ctx context.Context, rel string, target interface{}) error {
	body, err := c.GetBytes(ctx, rel)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

func (c Client) GetBytes(ctx context.Context, rel string) ([]byte, error) {
	u, err := c.URL(rel)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("GET %s failed: %s", u, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func (c Client) URL(rel string) (string, error) {
	base, err := url.Parse(c.BaseURL + "/")
	if err != nil {
		return "", err
	}
	ref, err := url.Parse(strings.TrimLeft(rel, "/"))
	if err != nil {
		return "", err
	}
	return base.ResolveReference(ref).String(), nil
}

func (c Client) client() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}
