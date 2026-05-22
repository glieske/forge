package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type updateIndex struct {
	Schema           int      `json:"schema"`
	Channel          string   `json:"channel"`
	Latest           string   `json:"latest"`
	MinimumSupported string   `json:"minimum_supported"`
	Versions         []string `json:"versions"`
}

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: update-release-index <index.json> <channel> <version>")
		os.Exit(2)
	}
	path := os.Args[1]
	channel := os.Args[2]
	version := os.Args[3]

	idx := updateIndex{
		Schema:           1,
		Channel:          channel,
		Latest:           version,
		MinimumSupported: "0.1.0",
	}
	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &idx); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else if err != nil && !os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	idx.Schema = 1
	idx.Channel = channel
	idx.Latest = version
	if idx.MinimumSupported == "" {
		idx.MinimumSupported = "0.1.0"
	}
	idx.Versions = prependUnique(version, idx.Versions)

	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func prependUnique(version string, versions []string) []string {
	out := []string{version}
	for _, existing := range versions {
		if existing != version {
			out = append(out, existing)
		}
	}
	return out
}
