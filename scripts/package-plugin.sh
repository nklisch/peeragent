#!/usr/bin/env sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
PLUGIN="$ROOT/plugin"

# Regenerate the curated tree WITHOUT removing committed platform binaries.
# plugin/bin/<goos>-<goarch>/peeragent are owned by .github/workflows/build-binaries.yml.
rm -rf "$PLUGIN/.claude-plugin" "$PLUGIN/.codex-plugin" "$PLUGIN/skills"
rm -f "$PLUGIN/.mcp.claude.json" "$PLUGIN/.mcp.codex.json" "$PLUGIN/bin/peeragent"
mkdir -p "$PLUGIN/.claude-plugin" "$PLUGIN/.codex-plugin" "$PLUGIN/bin" "$PLUGIN/skills"

cp "$ROOT/.claude-plugin/plugin.json" "$PLUGIN/.claude-plugin/plugin.json"
cp "$ROOT/.codex-plugin/plugin.json" "$PLUGIN/.codex-plugin/plugin.json"
cp "$ROOT/.mcp.claude.json" "$PLUGIN/.mcp.claude.json"
cp "$ROOT/.mcp.codex.json" "$PLUGIN/.mcp.codex.json"
cp "$ROOT/bin/peeragent" "$PLUGIN/bin/peeragent"
chmod +x "$PLUGIN/bin/peeragent"

for skill in "$ROOT"/skills/*; do
  if [ -d "$skill" ]; then
    cp -R "$skill" "$PLUGIN/skills/"
  fi
done
