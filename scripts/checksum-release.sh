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

version_dir="$OUT_ROOT/forge/updates/$CHANNEL/$VERSION"
if [ ! -d "$version_dir" ]; then
  echo "release directory does not exist: $version_dir" >&2
  exit 2
fi

rm -f "$version_dir/checksums.txt" "$version_dir/checksums.txt.sig"
(cd "$version_dir" && shasum -a 256 forge_* > checksums.txt)
go run ./tools/sign-checksums "$version_dir/checksums.txt" "$version_dir/checksums.txt.sig"

echo "checksums written and signed in $version_dir"
