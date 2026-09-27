#!/usr/bin/env sh
set -eu

# Build the committed per-platform peeragent binaries into plugin/bin/<target>/.
# Single source of truth for the committed-binary build recipe, used by
# .github/workflows/build-binaries.yml (initial build + rebuild-after-rebase).
# Flags match scripts/release.sh so committed binaries and release archives are
# byte-identical given the same toolchain. -buildvcs=false keeps builds free of
# VCS provenance so a dirty/standalone tree does not change the output.

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
SIZE_BUDGET_BYTES=${SIZE_BUDGET_BYTES:-8388608} # 8 MB per binary

for pair in "linux amd64" "linux arm64" "darwin amd64" "darwin arm64" "windows amd64" "windows arm64"; do
  # shellcheck disable=SC2086
  set -- $pair
  goos=$1
  goarch=$2
  target="$goos-$goarch"
  dest="$ROOT/plugin/bin/$target"
  binary=peeragent
  [ "$goos" = windows ] && binary=peeragent.exe
  mkdir -p "$dest"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    go build -trimpath -buildvcs=false -ldflags="-s -w" -o "$dest/$binary" "$ROOT/cmd/peeragent"
  size=$(wc -c < "$dest/$binary")
  echo "$target: $size bytes"
  if [ "$size" -gt "$SIZE_BUDGET_BYTES" ]; then
    echo "ERROR: $target exceeds size budget ($size > $SIZE_BUDGET_BYTES bytes)" >&2
    exit 1
  fi
done
