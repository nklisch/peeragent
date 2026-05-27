#!/usr/bin/env sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

mkdir -p "$ROOT/dist"
go build -o "$ROOT/dist/peeragent" "$ROOT/cmd/peeragent"
