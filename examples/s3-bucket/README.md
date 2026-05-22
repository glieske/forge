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
    index.json
    <plugin-name>/
      index.json
      <semver>/
        manifest.toml
        forge-<plugin>_<os>_<arch>.tar.gz
        forge-<plugin>_<os>_<arch>.zip
        checksums.txt
        checksums.txt.sig
  updates/
    <channel>/
      index.json
      <semver>/
        forge_<os>_<arch>.tar.gz
        forge_<os>_<arch>.zip
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

`checksums.txt.sig` is a base64-encoded Ed25519 signature of the raw `checksums.txt` file. `forge` verifies the signature using `security.public_key` from config.

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

Generate a key pair with:

```sh
go run ./tools/generate-keypair
```

Store `FORGE_ED25519_PRIVATE_KEY` as a GitHub secret. Put `FORGE_ED25519_PUBLIC_KEY` in `security.public_key` for clients.

AWS credentials can be provided either by repository variable `AWS_ROLE_TO_ASSUME` for OIDC, or by secrets `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`.

The workflow uploads only update artifacts to:

```text
s3://<bucket>/<prefix>/updates/<channel>/
```

Plugin artifacts should be published by the plugin release process using the layout shown above.
