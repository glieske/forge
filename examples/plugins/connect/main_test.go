package main

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestCompletions(t *testing.T) {
	t.Setenv("FORGE_GLOBAL_CONFIG_JSON", `{"services":["db","redis"],"environments":["dev","stage"]}`)
	values := completions("service", "d")
	if len(values) != 1 || values[0] != "db" {
		t.Fatalf("values = %#v", values)
	}
}

func TestDescribeShape(t *testing.T) {
	t.Setenv("FORGE_PLUGIN_VERSION", "1.2.0")
	var buf bytes.Buffer
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	ok := handleForgeProtocol([]string{"--forge-describe"})
	_ = w.Close()
	os.Stdout = old
	if !ok {
		t.Fatal("protocol flag was not handled")
	}
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["name"] != "connect" {
		t.Fatalf("payload = %#v", payload)
	}
}
