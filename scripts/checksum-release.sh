#!/usr/bin/env sh
set -eu

CHANNEL="${CHANNEL:-stable}"
VERSION="${VERSION:-}"
OUT_ROOT="${OUT_ROOT:-dist/s3}"

if [ -z "$VERSION" ]; then
  echo "VERSION is required, for example VERSION=0.2.0" >&2
  exit 2
fi

if [ -z "${FORGE_ED25519_PRIVATE_KEY:-}" ]; then
  echo "FORGE_ED25519_PRIVATE_KEY is required to sign checksums.txt" >&2
  exit 2
fi
if [ -z "${FORGE_ED25519_PUBLIC_KEY:-}" ]; then
  echo "FORGE_ED25519_PUBLIC_KEY is required to publish the update repository public key" >&2
  exit 2
fi

version_dir="$OUT_ROOT/forge/updates/$CHANNEL/$VERSION"
channel_dir="$OUT_ROOT/forge/updates/$CHANNEL"
updates_dir="$OUT_ROOT/forge/updates"
if [ ! -d "$version_dir" ]; then
  echo "release directory does not exist: $version_dir" >&2
  exit 2
fi

rm -f "$version_dir/checksums.txt" "$version_dir/checksums.txt.sig"
(cd "$version_dir" && shasum -a 256 forge_* > checksums.txt)
go run ./tools/sign-checksums "$version_dir/checksums.txt" "$version_dir/checksums.txt.sig"
go run ./tools/sign-file "$channel_dir/index.json" "$channel_dir/index.json.sig"
printf "%s\n" "$FORGE_ED25519_PUBLIC_KEY" > "$updates_dir/public-key.ed25519"

echo "checksums and update index signed in $channel_dir"
