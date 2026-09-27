#!/usr/bin/env sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

mkdir -p "$ROOT/dist"
binary=peeragent
[ "$(go env GOOS)" = windows ] && binary=peeragent.exe
go build -o "$ROOT/dist/$binary" "$ROOT/cmd/peeragent"
