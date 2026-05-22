package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/glieske/forge/internal/platform"
)

type Description struct {
	Schema       int                 `json:"schema"`
	Name         string              `json:"name"`
	Version      string              `json:"version"`
	Summary      string              `json:"summary"`
	Capabilities []string            `json:"capabilities"`
	Commands     []CommandDescriptor `json:"commands"`
}

type CommandDescriptor struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Examples    []string `json:"examples"`
}

type ConfigSchema struct {
	Schema int          `json:"schema"`
	Fields []ConfigSpec `json:"fields"`
}

type CompletionResult struct {
	Schema int      `json:"schema"`
	Values []string `json:"values"`
}

func (m Manager) Describe(ctx context.Context, name string) (Description, error) {
	var out Description
	return out, m.queryJSON(ctx, name, []string{"--forge-describe"}, &out)
}

func (m Manager) ConfigSchema(ctx context.Context, name string) (ConfigSchema, error) {
	var out ConfigSchema
	return out, m.queryJSON(ctx, name, []string{"--forge-config-schema"}, &out)
}

func (m Manager) Complete(ctx context.Context, name, arg string, prefix string) (CompletionResult, error) {
	var out CompletionResult
	return out, m.queryJSON(ctx, name, []string{"--forge-complete", arg, prefix}, &out)
}

func (m Manager) queryJSON(ctx context.Context, name string, args []string, target interface{}) error {
	item, err := m.Info(name)
	if err != nil {
		return err
	}
	if err := m.requireTrusted(item); err != nil {
		return err
	}
	qctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	exe := filepath.Join(item.Current, platform.ExeName(item.Manifest.Entrypoint))
	cmd := exec.CommandContext(qctx, exe, args...)
	cmd.Env = m.pluginEnv(item)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	data, err := cmd.Output()
	if err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("%w: %s", err, stderr.String())
		}
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return err
	}
	return nil
}
