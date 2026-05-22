#!/usr/bin/env sh
set -eu

build_one() {
  goos="$1"
  goarch="$2"
  out="bin/forge_${goos}_${goarch}"
  if [ "$goos" = "windows" ]; then
    out="$out.exe"
  fi
  echo "building $goos/$goarch"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build -o "$out" ./cmd/forge
}

build_one linux amd64
build_one linux arm64
build_one darwin amd64
build_one darwin arm64
build_one windows amd64
