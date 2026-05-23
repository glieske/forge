# S3 Bucket Layout

This directory documents the static HTTPS layout expected by `forge`.

Publish the contents below to an S3 bucket or any static HTTP host. Configure the URLs with:

```sh
forge config set repositories.plugins_url https://bucket.example.com/forge/plugins
forge config set repositories.updates_url https://bucket.example.com/forge/updates
forge self-update channel stable
```

## Required Layout

```text
forge/
  plugins/
    public-key.ed25519
    index.json
    index.json.sig
    <plugin-name>/
      index.json
      index.json.sig
      <semver>/
        manifest.toml
        manifest.toml.sig
        forge-<plugin>_<os>_<arch>.tar.gz
        forge-<plugin>_<os>_<arch>.zip
        checksums.txt
        checksums.txt.sig
  updates/
    public-key.ed25519
    <channel>/
      index.json
      index.json.sig
      <semver>/
        forge_<os>_<arch>.tar.gz
        forge_<os>_<arch>.zip
        forge_<os>_<arch>.tar.gz.sigstore.json
        forge_<os>_<arch>.zip.sigstore.json
        checksums.txt
        checksums.txt.sig
```

Use `.tar.gz` for Linux/macOS and `.zip` for Windows.

Supported OS/arch names follow Go conventions, for example:

- `linux_amd64`
- `linux_arm64`
- `darwin_amd64`
- `darwin_arm64`
- `windows_amd64`

## Checksums And Signatures

Every package must be listed in `checksums.txt`:

```text
<sha256>  forge-connect_darwin_arm64.tar.gz
<sha256>  forge-connect_linux_amd64.tar.gz
```

`public-key.ed25519` contains the base64-encoded Ed25519 public key used to verify repository signatures. `security.public_key` in local config is optional; when set, it overrides repository keys.

Every metadata file read from the repository must have a sibling Ed25519 signature:

- `plugins/index.json.sig`
- `plugins/<plugin>/index.json.sig`
- `plugins/<plugin>/<version>/manifest.toml.sig`
- `updates/<channel>/index.json.sig`

`checksums.txt.sig` is a base64-encoded Ed25519 signature of the raw `checksums.txt` file. `forge` verifies signatures using `security.public_key` from config, or `public-key.ed25519` from the relevant repository root when the config value is empty.

When repository keys are loaded from S3, `forge` stores the accepted key fingerprint in local `trusted-repositories.toml`. Later key changes are rejected until the user verifies the new key and resets the local trust record.

With default security settings, `forge` records verified signature metadata during installation and refuses to run plugins that do not have trust metadata. If `security.public_key` is configured later, already installed plugins must match that key fingerprint to run.

## Notes

The sample files in this directory are templates. Replace placeholder checksums, signatures, versions, and bucket URLs during release publishing.

## Plugin Dependencies

Plugin manifests may declare dependencies on other plugins:

```toml
[[dependencies]]
name = "aws"
version = ">=1.0.0 <2.0.0"
channel = "stable"
optional = true
```

`version` uses SemVer constraints from `Masterminds/semver`. If `channel` is omitted, the active install channel is used. Required dependencies are installed first; optional dependencies are skipped when missing or incompatible.

## GitHub Actions Deployment

The `CI` workflow can publish update artifacts through `workflow_dispatch`.

Required manual inputs:

- `deploy`: set to `true`.
- `s3_bucket`: bucket name without `s3://`.
- `version`: version to publish, for example `0.2.0`.
- `channel`: `stable`, `beta`, or `dev`.
- `s3_prefix`: defaults to `forge`.

Required secret:

- `FORGE_ED25519_PRIVATE_KEY`: base64-encoded Ed25519 seed or private key used to sign `checksums.txt`.
- `FORGE_ED25519_PUBLIC_KEY`: base64-encoded Ed25519 public key published as `updates/public-key.ed25519`.

Generate a key pair with:

```sh
go run ./tools/generate-keypair
```

Store `FORGE_ED25519_PRIVATE_KEY` as a GitHub secret. Publish `FORGE_ED25519_PUBLIC_KEY` as `forge/plugins/public-key.ed25519` and `forge/updates/public-key.ed25519`; clients may optionally set the same value as `security.public_key` to pin the key locally.

AWS credentials can be provided either by repository secret `AWS_ROLE_TO_ASSUME` for OIDC, or by secrets `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`.

The workflow uploads only update artifacts to:

```text
s3://<bucket>/<prefix>/updates/<channel>/
```

Plugin artifacts should be published by the plugin release process using the layout shown above.

The same deploy job creates or updates GitHub Release `v<version>` and uploads the version assets: platform archives, Cosign bundles, `checksums.txt`, and `checksums.txt.sig`. Non-`stable` channels are marked as prereleases.

Release artifacts are additionally signed with Cosign keyless signing in GitHub Actions. The workflow writes Sigstore bundles next to each archive, then calculates and signs final checksums:

```text
forge_linux_amd64.tar.gz
forge_linux_amd64.tar.gz.sigstore.json
```

Verify a downloaded release artifact with:

```sh
cosign verify-blob forge_linux_amd64.tar.gz \
  --bundle forge_linux_amd64.tar.gz.sigstore.json \
  --certificate-identity=https://github.com/glieske/forge/.github/workflows/ci.yml@refs/heads/main \
  --certificate-oidc-issuer=https://token.actions.githubusercontent.com
```
