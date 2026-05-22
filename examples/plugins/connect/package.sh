#!/usr/bin/env sh
set -eu

VERSION="${VERSION:-1.2.0}"
OUT_DIR="${OUT_DIR:-../../s3-bucket/forge/plugins/connect/$VERSION}"

mkdir -p "$OUT_DIR"
cp manifest.toml "$OUT_DIR/manifest.toml"

build_one() {
  goos="$1"
  goarch="$2"
  work="dist/${goos}_${goarch}"
  rm -rf "$work"
  mkdir -p "$work"

  exe="forge-connect"
  if [ "$goos" = "windows" ]; then
    exe="forge-connect.exe"
  fi

  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build -o "$work/$exe" .

  if [ "$goos" = "windows" ]; then
    (cd "$work" && zip -q "../../../$OUT_DIR/forge-connect_${goos}_${goarch}.zip" "$exe")
  else
    (cd "$work" && tar -czf "../../../$OUT_DIR/forge-connect_${goos}_${goarch}.tar.gz" "$exe")
  fi
}

build_one linux amd64
build_one linux arm64
build_one darwin amd64
build_one darwin arm64
build_one windows amd64

(cd "$OUT_DIR" && shasum -a 256 forge-connect_* > checksums.txt)
echo "Package files written to $OUT_DIR"
echo "Sign $OUT_DIR/checksums.txt and write checksums.txt.sig before publishing."
