#!/usr/bin/env sh
set -eu

VERSION="${VERSION:-}"
CHANNEL="${CHANNEL:-stable}"
COMMIT="${GITHUB_SHA:-$(git rev-parse --short HEAD 2>/dev/null || echo unknown)}"
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
rm -rf "$version_dir"
mkdir -p "$version_dir"

build_one() {
  goos="$1"
  goarch="$2"
  exe="forge"
  if [ "$goos" = "windows" ]; then
    exe="forge.exe"
  fi

  work="dist/build/${goos}_${goarch}"
  rm -rf "$work"
  mkdir -p "$work"

  echo "building $goos/$goarch"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build \
    -ldflags "-s -w -X main.version=$VERSION -X main.commit=$COMMIT" \
    -o "$work/$exe" ./cmd/forge

  if [ "$goos" = "windows" ]; then
    package="forge_${goos}_${goarch}.zip"
    (cd "$work" && zip -q "../../s3/forge/updates/$CHANNEL/$VERSION/$package" "$exe")
  else
    package="forge_${goos}_${goarch}.tar.gz"
    (cd "$work" && tar -czf "../../s3/forge/updates/$CHANNEL/$VERSION/$package" "$exe")
  fi
}

build_one linux amd64
build_one linux arm64
build_one darwin amd64
build_one darwin arm64
build_one windows amd64

(cd "$version_dir" && shasum -a 256 forge_* > checksums.txt)
go run ./tools/sign-checksums "$version_dir/checksums.txt" "$version_dir/checksums.txt.sig"

channel_dir="$OUT_ROOT/forge/updates/$CHANNEL"
mkdir -p "$channel_dir"
cat > "$channel_dir/index.json" <<EOF
{
  "schema": 1,
  "channel": "$CHANNEL",
  "latest": "$VERSION",
  "minimum_supported": "0.1.0",
  "versions": [
    "$VERSION"
  ]
}
EOF

echo "release repository written to $OUT_ROOT/forge/updates/$CHANNEL"
