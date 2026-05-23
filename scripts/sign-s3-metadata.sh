#!/usr/bin/env sh
set -eu

ROOT="${ROOT:-dist/s3/forge}"

if [ -z "${FORGE_ED25519_PRIVATE_KEY:-}" ]; then
  echo "FORGE_ED25519_PRIVATE_KEY is required" >&2
  exit 2
fi
if [ -z "${FORGE_ED25519_PUBLIC_KEY:-}" ]; then
  echo "FORGE_ED25519_PUBLIC_KEY is required" >&2
  exit 2
fi

sign_file() {
  file="$1"
  if [ -f "$file" ]; then
    echo "signing $file"
    go run ./tools/sign-file "$file" "$file.sig"
  fi
}

sign_many() {
  find "$1" -type f -name "$2" -print | while IFS= read -r file; do
    sign_file "$file"
  done
}

plugins_dir="$ROOT/plugins"
updates_dir="$ROOT/updates"

if [ -d "$plugins_dir" ]; then
  printf "%s\n" "$FORGE_ED25519_PUBLIC_KEY" > "$plugins_dir/public-key.ed25519"
  sign_many "$plugins_dir" "index.json"
  sign_many "$plugins_dir" "manifest.toml"
  sign_many "$plugins_dir" "checksums.txt"
fi

if [ -d "$updates_dir" ]; then
  printf "%s\n" "$FORGE_ED25519_PUBLIC_KEY" > "$updates_dir/public-key.ed25519"
  sign_many "$updates_dir" "index.json"
  sign_many "$updates_dir" "checksums.txt"
fi

echo "S3 repository metadata signed under $ROOT"
