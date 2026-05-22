package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type globals map[string][]string

func main() {
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
