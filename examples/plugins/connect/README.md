# Example Plugin: connect

This is a minimal external `forge` plugin. It builds a standalone binary named `forge-connect` and is installed by `forge` as the `connect` subcommand.

Build locally:

```sh
go build -o forge-connect .
```

Run directly:

```sh
FORGE_PLUGIN_NAME=connect \
FORGE_PLUGIN_VERSION=1.2.0 \
FORGE_CONFIG_PATH=/tmp/forge/config.toml \
FORGE_DATA_DIR=/tmp/forge/data \
FORGE_PLUGIN_DIR=/tmp/forge/data/plugins/connect \
FORGE_SECRETS_MODE=file \
FORGE_GLOBAL_CONFIG_JSON='{"environments":["dev","stage","prod"],"services":["db","redis"]}' \
./forge-connect db stage
```

When installed through `forge`, users run:

```sh
forge connect db stage
```

The matching manifest template is `manifest.toml`.
