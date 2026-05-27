#!/usr/bin/env sh
set -eu

# scripts/bump.sh — bump version, repackage, validate, commit, tag, push
#
# Usage:
#   scripts/bump.sh <new-version>
#   scripts/bump.sh <new-version> --message "Add peer-review skill"
#   scripts/bump.sh <new-version> --no-push
#
# Edits the two plugin manifests and README release examples, resyncs the
# packaged plugin/ tree, runs validation, then commits the changes, tags
# v<new-version>, and pushes the branch and tag to origin.
#
# Refuses to run with a dirty working tree, an existing tag for the target
# version, or the current version equal to the target — so re-runs surface
# as errors instead of empty commits.

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

PUSH=1
MESSAGE=""
VERSION=""

usage() {
  cat <<'EOF'
usage: scripts/bump.sh <new-version> [--message <msg>] [--no-push]

Bumps the version across both plugin manifests and the README release
examples, repackages plugin/, runs validation, then commits, tags
v<new-version>, and pushes branch + tag to origin.

Options:
  -m, --message <msg>   Commit message (default: "Bump to v<version>")
      --no-push         Commit and tag locally, skip git push
  -h, --help            Show this help

Examples:
  scripts/bump.sh 0.1.5
  scripts/bump.sh 0.2.0 --message "Add peer-review skill"
  scripts/bump.sh 0.1.5 --no-push
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    -h|--help)
      usage; exit 0 ;;
    --no-push)
      PUSH=0; shift ;;
    -m|--message)
      if [ $# -lt 2 ]; then echo "$1 requires an argument" >&2; exit 2; fi
      MESSAGE=$2; shift 2 ;;
    -*)
      echo "unknown option: $1" >&2; usage >&2; exit 2 ;;
    *)
      if [ -n "$VERSION" ]; then
        echo "unexpected positional argument: $1" >&2; usage >&2; exit 2
      fi
      VERSION=$1; shift ;;
  esac
done

if [ -z "$VERSION" ]; then
  echo "new-version is required" >&2; usage >&2; exit 2
fi

VERSION=${VERSION#v}

case "$VERSION" in
  [0-9]*.[0-9]*.[0-9]*) ;;
  *) echo "version must be MAJOR.MINOR.PATCH (got '$VERSION')" >&2; exit 2 ;;
esac

[ -z "$MESSAGE" ] && MESSAGE="Bump to v$VERSION"

cd "$ROOT"

CURRENT=$(awk -F'"' '/"version"[[:space:]]*:/ { print $4; exit }' \
  .claude-plugin/plugin.json)

if [ -z "$CURRENT" ]; then
  echo "could not read current version from .claude-plugin/plugin.json" >&2
  exit 2
fi

if [ "$CURRENT" = "$VERSION" ]; then
  echo "version already at $VERSION; nothing to bump" >&2
  exit 2
fi

if [ -n "$(git status --porcelain)" ]; then
  echo "working tree has uncommitted changes; commit or stash first" >&2
  exit 2
fi

if git rev-parse "v$VERSION" >/dev/null 2>&1; then
  echo "tag v$VERSION already exists" >&2
  exit 2
fi

echo "==> bumping $CURRENT -> $VERSION"

# Portable in-place edit: run awk, write to tmp, rename over original.
edit_in_place() {
  _file=$1
  shift
  _tmp=$(mktemp)
  awk "$@" "$_file" > "$_tmp"
  mv "$_tmp" "$_file"
}

# Rewrite the first "version": "X.Y.Z" line in a plugin manifest.
bump_manifest() {
  edit_in_place "$1" -v v="$VERSION" '
    /"version"[[:space:]]*:/ && !done {
      sub(/"[0-9]+\.[0-9]+\.[0-9]+"/, "\"" v "\"")
      done = 1
    }
    { print }
  '
}

# Replace every literal occurrence of $CURRENT with $VERSION in a file
# (used for README release examples; literal replacement avoids regex
# escaping of the dots in version strings).
bump_text() {
  edit_in_place "$1" -v old="$CURRENT" -v new="$VERSION" '
    {
      out = ""
      s = $0
      ol = length(old)
      while ((i = index(s, old)) > 0) {
        out = out substr(s, 1, i-1) new
        s = substr(s, i + ol)
      }
      print out s
    }
  '
}

echo "==> editing manifests"
bump_manifest .claude-plugin/plugin.json
bump_manifest .codex-plugin/plugin.json

echo "==> editing README release examples"
bump_text README.md

echo "==> repackaging plugin/"
scripts/package-plugin.sh

echo "==> validating"
scripts/validate.sh

echo "==> committing"
git add \
  .claude-plugin/plugin.json \
  .codex-plugin/plugin.json \
  plugin/.claude-plugin/plugin.json \
  plugin/.codex-plugin/plugin.json \
  README.md
git commit -m "$MESSAGE"

echo "==> tagging v$VERSION"
git tag "v$VERSION"

if [ "$PUSH" -eq 1 ]; then
  branch=$(git rev-parse --abbrev-ref HEAD)
  echo "==> pushing $branch and v$VERSION"
  git push origin "$branch"
  git push origin "v$VERSION"
else
  echo "==> --no-push set; skipping git push"
fi

echo "==> bump complete: v$VERSION"
