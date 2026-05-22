package plugin

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Manifest struct {
	Schema          int           `toml:"schema"`
	Name            string        `toml:"name"`
	Version         string        `toml:"version"`
	Description     string        `toml:"description"`
	Entrypoint      string        `toml:"entrypoint"`
	MinForgeVersion string        `toml:"min_forge_version"`
	Commands        []CommandSpec `toml:"commands"`
	Config          []ConfigSpec  `toml:"config"`
}

type CommandSpec struct {
	Name        string    `toml:"name"`
	Description string    `toml:"description"`
	Args        []ArgSpec `toml:"args"`
}

type ArgSpec struct {
	Name           string   `toml:"name"`
	Position       int      `toml:"position"`
	Required       bool     `toml:"required"`
	Description    string   `toml:"description"`
	ValueProvider  string   `toml:"value_provider"`
	FallbackValues []string `toml:"fallback_values"`
}

type ConfigSpec struct {
	Key           string   `toml:"key"`
	Type          string   `toml:"type"`
	Description   string   `toml:"description"`
	ValueProvider string   `toml:"value_provider"`
	Secret        bool     `toml:"secret"`
	Required      bool     `toml:"required"`
	Values        []string `toml:"values"`
}

func DecodeManifest(data []byte) (Manifest, error) {
	var m Manifest
	if err := toml.Unmarshal(data, &m); err != nil {
		return Manifest{}, err
	}
	if err := m.Validate(); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	return DecodeManifest(data)
}

func (m Manifest) Validate() error {
	if m.Schema == 0 {
		return fmt.Errorf("manifest schema is required")
	}
	if m.Name == "" {
		return fmt.Errorf("manifest name is required")
	}
	if m.Version == "" {
		return fmt.Errorf("manifest version is required")
	}
	if m.Entrypoint == "" {
		return fmt.Errorf("manifest entrypoint is required")
	}
	for _, cmd := range m.Commands {
		if cmd.Name == "" {
			return fmt.Errorf("command name is required")
		}
	}
	return nil
}
