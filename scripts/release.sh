#!/usr/bin/env sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
PUBLISH=0

if [ "${1:-}" = "--publish" ]; then
  PUBLISH=1
  shift
fi

VERSION=${1:-}
if [ -z "$VERSION" ]; then
  VERSION=$(awk -F'\"' '/\"version\"[[:space:]]*:/ { print $4; exit }' "$ROOT/.claude-plugin/plugin.json")
fi
VERSION=${VERSION#v}
VERSION=${VERSION%%+*}

if [ -z "$VERSION" ]; then
  echo "release version is required" >&2
  exit 2
fi

OUT="$ROOT/dist/release"
BUILD="$OUT/build"
rm -rf "$OUT"
mkdir -p "$BUILD"

build_target() {
  goos=$1
  goarch=$2
  target="$goos-$goarch"
  dir="$BUILD/$target"
  mkdir -p "$dir"
  echo "building $target"
  CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build -trimpath -ldflags="-s -w" -o "$dir/alt-subagent" "$ROOT/cmd/alt-subagent"
  tar -C "$dir" -czf "$OUT/alt-subagent_${VERSION}_${goos}_${goarch}.tar.gz" alt-subagent
}

build_target linux amd64
build_target linux arm64
build_target darwin amd64
build_target darwin arm64

(
  cd "$OUT"
  sha256sum alt-subagent_*.tar.gz > checksums.txt
)

rm -rf "$BUILD"

echo "release artifacts written to $OUT"

if [ "$PUBLISH" -eq 1 ]; then
  if ! command -v gh >/dev/null 2>&1; then
    echo "gh is required for --publish" >&2
    exit 2
  fi
  tag="v$VERSION"
  gh release create "$tag" "$OUT"/*     --target "${GITHUB_SHA:-HEAD}"     --title "alt-subagent $tag"     --notes "Release binaries for alt-subagent $tag. Marketplace installs can fetch these assets on first use."
fi
