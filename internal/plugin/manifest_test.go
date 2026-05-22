package plugin

import "testing"

func TestDecodeManifest(t *testing.T) {
	m, err := DecodeManifest([]byte(`
schema = 1
name = "connect"
version = "1.2.0"
description = "Open tunnels"
entrypoint = "forge-connect"

[[dependencies]]
name = "aws"
version = ">=1.0.0 <2.0.0"
channel = "stable"
optional = true

[[commands]]
name = "connect"
description = "Connect"

[[commands.args]]
name = "service"
position = 1
required = true
value_provider = "global.services"
fallback_values = ["db"]
`))
	if err != nil {
		t.Fatal(err)
	}
	if m.Commands[0].Args[0].ValueProvider != "global.services" {
		t.Fatalf("unexpected provider: %s", m.Commands[0].Args[0].ValueProvider)
	}
	if len(m.Dependencies) != 1 || m.Dependencies[0].Name != "aws" {
		t.Fatalf("dependencies = %#v", m.Dependencies)
	}
}
