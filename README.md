# forge

[![CI](https://github.com/glieske/forge/actions/workflows/ci.yml/badge.svg)](https://github.com/glieske/forge/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.26.2-00ADD8?logo=go)](https://go.dev/)
[![Go Report Card](https://goreportcard.com/badge/github.com/glieske/forge)](https://goreportcard.com/report/github.com/glieske/forge)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

`forge` is a developer-focused CLI/TUI tool that can be extended with external plugins. The core binary ships without plugins; teams publish plugin manifests and platform-specific binaries to a static S3 bucket, and users install new subcommands on demand.

Example: after installing a `connect` plugin, developers can run:

```sh
forge connect db stage
```

The plugin itself is a separate executable (`forge-connect`) managed by `forge`.

## Features

- CLI and terminal UI for Linux, macOS, and Windows.
- Plugin discovery, installation, update, removal, and command routing.
- Plugin-to-plugin dependencies with SemVer constraints.
- Static HTTPS/S3 plugin repository, no AWS SDK required.
- Self-update repository with `stable`, `beta`, and `dev` channels.
- TOML global configuration shared with plugins.
- Secret storage through system keychain/keyring with file fallback.
- SHA256 checksum and Ed25519 signature verification for repository metadata and downloaded artifacts.
- Refuses to run plugins that were not installed from a signature verified by a known public key.
- Fuzzy command and argument selection in the TUI.

## Installation

The project currently builds from source:

```sh
git clone git@github.com:glieske/forge.git
cd forge
make build
./bin/forge --version
```

Before plugin discovery or self-update can work, configure repository URLs:

```sh
forge config set repositories.plugins_url https://bucket.example.com/forge/plugins
forge config set repositories.updates_url https://bucket.example.com/forge/updates
```

By default, signature verification loads the trusted Ed25519 public key from each repository root:

```text
<plugins_url>/public-key.ed25519
<updates_url>/public-key.ed25519
```

When `security.public_key` is empty, `forge` pins the first accepted repository key fingerprint in `trusted-repositories.toml` and rejects later key changes until the user verifies and resets that trust record.

Optionally pin/override that key locally:

```sh
forge config set security.public_key <base64-ed25519-public-key>
```

For local development, run commands through:

```sh
./scripts/dev.sh plugin list
./scripts/dev.sh config get repositories.channel
```

## Basic Usage

Open the TUI:

```sh
forge
```

The TUI includes plugin browsing, command selection, global config editing, plugin-specific settings generated from manifests, and secret management. If repository URLs are missing, the dashboard shows configuration warnings.

List and install plugins:

```sh
forge plugin available
forge plugin install connect
forge plugin list
```

Run an installed plugin:

```sh
forge connect db stage
```

Manage configuration:

```sh
forge config get repositories.plugins_url
forge config set repositories.plugins_url https://bucket.example.com/forge/plugins
forge config set repositories.updates_url https://bucket.example.com/forge/updates
forge self-update channel stable
```

Manage secrets:

```sh
forge secret set global token
forge secret get global token
forge secret delete global token
```

## Plugin Repository

`forge` expects a static bucket layout:

```text
forge/
  plugins/
    public-key.ed25519
    index.json
    index.json.sig
    connect/
      index.json
      index.json.sig
      1.2.0/
        manifest.toml
        manifest.toml.sig
        forge-connect_linux_amd64.tar.gz
        checksums.txt
        checksums.txt.sig
  updates/
    public-key.ed25519
    stable/
      index.json
      index.json.sig
      0.2.0/
        forge_linux_amd64.tar.gz
        checksums.txt
        checksums.txt.sig
```

See [examples/s3-bucket](examples/s3-bucket/README.md) for the full structure.

## Example Plugin

A minimal `connect` plugin is available in [examples/plugins/connect](examples/plugins/connect/README.md).

Build it directly:

```sh
cd examples/plugins/connect
go build -o forge-connect .
./forge-connect db stage
```

Package plugin artifacts for the sample S3 layout:

```sh
VERSION=1.2.0 ./package.sh
```

Plugins can declare dependencies in `manifest.toml`:

```toml
[[dependencies]]
name = "aws"
version = ">=1.0.0 <2.0.0"
channel = "stable"
optional = true
```

Required dependencies are installed before the requested plugin. Optional dependencies are used when already present or skipped when unavailable.

Plugins may also implement an optional JSON-over-stdout protocol:

```sh
forge-connect --forge-describe
forge-connect --forge-config-schema
forge-connect --forge-complete service d
```

This gives `forge` richer descriptions, config schemas, and completions while keeping plugins as ordinary executables. No RPC runtime is required.

## Development

Common commands:

```sh
make fmt
make ci
make test
make build
make build-all
```

Generate an Ed25519 key pair for signing repository metadata and release checksums:

```sh
go run ./tools/generate-keypair
```

Build self-update artifacts locally:

```sh
VERSION=0.2.0 CHANNEL=stable make package-release
```

In GitHub Actions, release order is: build archives, sign archives with Cosign keyless OIDC, then calculate final SHA256 checksums, sign `checksums.txt`, sign the update `index.json`, and publish `updates/public-key.ed25519`.

## CI And Security

GitHub Actions runs formatting checks, `go vet`, tests, cross-compilation, `golangci-lint`, Trivy vulnerability scans, and S3 example validation. Manual workflow dispatch can publish update artifacts to a provided S3 bucket. Release archives are signed with Cosign keyless signing through GitHub OIDC; after Cosign bundles are written, final checksums and repository metadata are signed with the configured Ed25519 key.

Verify a Cosign bundle:

```sh
cosign verify-blob forge_linux_amd64.tar.gz \
  --bundle forge_linux_amd64.tar.gz.sigstore.json \
  --certificate-identity=https://github.com/glieske/forge/.github/workflows/ci.yml@refs/heads/main \
  --certificate-oidc-issuer=https://token.actions.githubusercontent.com
```

Dependabot monitors Go modules and GitHub Actions.

## License

Apache License 2.0. See [LICENSE](LICENSE).
