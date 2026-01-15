#!/usr/bin/env bash
set -euo pipefail

APP_NAME="GdbAlarm"
OUTPUT_DIR=.
GOFLAGS=${GOFLAGS:-}

build() {
  GOOS=$1 GOARCH=$2 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$OUTPUT_DIR/${APP_NAME}-$3" ./
}

build linux amd64 linux-amd64
GOARM=7 build linux arm linux-arm
build darwin arm64 darwin-arm64

echo "Built binaries in $OUTPUT_DIR"
