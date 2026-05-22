#!/usr/bin/env sh
set -eu

CHANNEL="${CHANNEL:-stable}"
VERSION="${VERSION:-}"
OUT_ROOT="${OUT_ROOT:-dist/s3}"

if [ -z "$VERSION" ]; then
  echo "VERSION is required, for example VERSION=0.2.0" >&2
  exit 2
fi

version_dir="$OUT_ROOT/forge/updates/$CHANNEL/$VERSION"
if [ ! -d "$version_dir" ]; then
  echo "release directory does not exist: $version_dir" >&2
  exit 2
fi

for artifact in "$version_dir"/forge_*; do
  case "$artifact" in
    *.tar.gz|*.zip)
      echo "cosign signing $artifact"
      cosign sign-blob "$artifact" --bundle "$artifact.sigstore.json" --yes
      ;;
  esac
done

echo "cosign bundles written to $version_dir"
