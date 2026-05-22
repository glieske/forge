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
- Static HTTPS/S3 plugin repository, no AWS SDK required.
- Self-update repository with `stable`, `beta`, and `dev` channels.
- TOML global configuration shared with plugins.
- Secret storage through system keychain/keyring with file fallback.
- SHA256 checksum and Ed25519 signature verification for downloaded artifacts.
- Fuzzy command and argument selection in the TUI.

## Installation

The project currently builds from source:

```sh
git clone git@github.com:glieske/forge.git
cd forge
make build
./bin/forge --version
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
    index.json
    connect/
      index.json
      1.2.0/
        manifest.toml
        forge-connect_linux_amd64.tar.gz
        checksums.txt
        checksums.txt.sig
  updates/
    stable/
      index.json
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

## Development

Common commands:

```sh
make fmt
make ci
make test
make build
make build-all
```

Generate an Ed25519 key pair for signing release checksums:

```sh
go run ./tools/generate-keypair
```

Build signed self-update artifacts:

```sh
VERSION=0.2.0 \
CHANNEL=stable \
FORGE_ED25519_PRIVATE_KEY=<base64-private-key> \
make package-release
```

## CI And Security

GitHub Actions runs formatting checks, `go vet`, tests, cross-compilation, `golangci-lint`, Trivy vulnerability scans, and S3 example validation. Manual workflow dispatch can publish signed update artifacts to a provided S3 bucket.

Dependabot monitors Go modules and GitHub Actions.

## License

Apache License 2.0. See [LICENSE](LICENSE).
