#!/usr/bin/env sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
PLUGIN="$ROOT/plugin"

rm -rf "$PLUGIN"
mkdir -p "$PLUGIN/.claude-plugin" "$PLUGIN/.codex-plugin" "$PLUGIN/bin" "$PLUGIN/skills"

cp "$ROOT/.claude-plugin/plugin.json" "$PLUGIN/.claude-plugin/plugin.json"
cp "$ROOT/.codex-plugin/plugin.json" "$PLUGIN/.codex-plugin/plugin.json"
cp "$ROOT/bin/peeragent" "$PLUGIN/bin/peeragent"
chmod +x "$PLUGIN/bin/peeragent"

for skill in "$ROOT"/skills/*; do
  if [ -d "$skill" ]; then
    cp -R "$skill" "$PLUGIN/skills/"
  fi
done
