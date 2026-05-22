package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type globals map[string][]string

func main() {
	if handleForgeProtocol(os.Args[1:]) {
		return
	}
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: forge-connect <service> <environment>")
		os.Exit(2)
	}

	service := os.Args[1]
	environment := os.Args[2]
	values := readGlobals()

	if !allowed(values["services"], service) {
		fmt.Fprintf(os.Stderr, "unknown service %q; known services: %s\n", service, strings.Join(values["services"], ", "))
		os.Exit(2)
	}
	if !allowed(values["environments"], environment) {
		fmt.Fprintf(os.Stderr, "unknown environment %q; known environments: %s\n", environment, strings.Join(values["environments"], ", "))
		os.Exit(2)
	}

	fmt.Printf("plugin=%s version=%s\n", env("FORGE_PLUGIN_NAME", "connect"), env("FORGE_PLUGIN_VERSION", "dev"))
	fmt.Printf("opening tunnel: service=%s environment=%s\n", service, environment)
	fmt.Printf("config=%s\n", env("FORGE_CONFIG_PATH", ""))
	fmt.Printf("plugin_dir=%s\n", env("FORGE_PLUGIN_DIR", ""))
}

func handleForgeProtocol(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "--forge-describe":
		writeJSON(map[string]any{
			"schema":       1,
			"name":         "connect",
			"version":      env("FORGE_PLUGIN_VERSION", "dev"),
			"summary":      "Open developer connectivity tunnels",
			"capabilities": []string{"describe", "config-schema", "complete"},
			"commands": []map[string]any{
				{
					"name":        "connect",
					"description": "Open a tunnel to an internal service",
					"examples":    []string{"forge connect db stage", "forge connect redis dev"},
				},
			},
		})
		return true
	case "--forge-config-schema":
		writeJSON(map[string]any{
			"schema": 1,
			"fields": []map[string]any{
				{
					"key":            "default_profile",
					"type":           "string",
					"description":    "Default AWS profile",
					"value_provider": "global.aws_profiles",
					"secret":         false,
					"required":       false,
				},
				{
					"key":         "api_token",
					"type":        "secret_string",
					"description": "Optional provider token",
					"secret":      true,
					"required":    false,
				},
			},
		})
		return true
	case "--forge-complete":
		arg := ""
		prefix := ""
		if len(args) > 1 {
			arg = args[1]
		}
		if len(args) > 2 {
			prefix = args[2]
		}
		writeJSON(map[string]any{
			"schema": 1,
			"values": completions(arg, prefix),
		})
		return true
	default:
		return false
	}
}

func completions(arg, prefix string) []string {
	values := readGlobals()
	var candidates []string
	switch arg {
	case "service":
		candidates = values["services"]
	case "environment":
		candidates = values["environments"]
	}
	if len(candidates) == 0 {
		return nil
	}
	out := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, prefix) {
			out = append(out, candidate)
		}
	}
	return out
}

func writeJSON(value any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func readGlobals() globals {
	raw := os.Getenv("FORGE_GLOBAL_CONFIG_JSON")
	if raw == "" {
		return globals{
			"services":     {"db", "redis"},
			"environments": {"dev", "stage", "prod"},
		}
	}
	var values globals
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to parse FORGE_GLOBAL_CONFIG_JSON: %v\n", err)
		return globals{}
	}
	return values
}

func allowed(values []string, value string) bool {
	if len(values) == 0 {
		return true
	}
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
