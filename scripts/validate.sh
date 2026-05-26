#!/usr/bin/env sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT"
VERSION=$(awk -F'"' '/"version"[[:space:]]*:/ { print $4; exit }' .claude-plugin/plugin.json)

step() {
  printf '\n==> %s\n' "$1"
}

step "go tests"
go test ./...

step "build"
scripts/build.sh
test -x dist/alt-subagent
test -x bin/alt-subagent

step "plugin package"
scripts/package-plugin.sh
test -x plugin/bin/alt-subagent
test -f plugin/.claude-plugin/plugin.json
test -f plugin/.codex-plugin/plugin.json
test -f plugin/skills/claude-implement/SKILL.md
test -f plugin/skills/codex-implement/SKILL.md
test -f plugin/skills/gemini-implement/SKILL.md

step "release artifacts"
scripts/release.sh "$VERSION"
test -f "dist/release/alt-subagent_${VERSION}_linux_amd64.tar.gz"
test -f "dist/release/alt-subagent_${VERSION}_linux_arm64.tar.gz"
test -f "dist/release/alt-subagent_${VERSION}_darwin_amd64.tar.gz"
test -f "dist/release/alt-subagent_${VERSION}_darwin_arm64.tar.gz"
test -f dist/release/checksums.txt

step "plugin metadata"
grep -q '"name": "alt-subagent"' .claude-plugin/plugin.json
grep -q '"name": "alt-subagent"' .codex-plugin/plugin.json
grep -q '"name": "alt-subagent"' .claude-plugin/marketplace.json
grep -q '"name": "alt-subagent"' .agents/plugins/marketplace.json
grep -q './plugin' .claude-plugin/marketplace.json
grep -q './plugin' .agents/plugins/marketplace.json
grep -q '"skills": "./skills/"' .codex-plugin/plugin.json
grep -q '"defaultPrompt"' .codex-plugin/plugin.json
grep -q 'name: codex-implement' skills/codex-implement/SKILL.md
grep -q 'allowed-tools: Bash' skills/codex-implement/SKILL.md
grep -q 'name: gemini-implement' skills/gemini-implement/SKILL.md
grep -q 'name: claude-implement' skills/claude-implement/SKILL.md

step "documentation examples"
grep -q 'make build' README.md
grep -q 'claude plugin marketplace add nklisch/alt-subagent' README.md
grep -q 'codex plugin marketplace add nklisch/alt-subagent' README.md
grep -q "make release VERSION=$VERSION" README.md
grep -q -- '--agent gemini' README.md
grep -q -- '--agent claude' README.md
grep -q -- '--effort high' README.md
grep -q -- '--effort xhigh' README.md
grep -q -- '--model opus' README.md
grep -q -- '--model gemini-3.5' README.md
grep -q -- '--async' README.md
grep -q -- '--status <job-id>' README.md
grep -q -- '--result <job-id>' README.md
grep -q -- '--cancel <job-id>' README.md
grep -q -- '--full-access' README.md

if grep -R -F -- '--status [job-id]' README.md docs skills; then
  echo "stale optional --status syntax found"
  exit 1
fi

if grep -R -F -- '--result [job-id]' README.md docs skills; then
  echo "stale optional --result syntax found"
  exit 1
fi

if grep -R -F -- '--effort low' README.md docs skills; then
  echo "unsupported low effort example found"
  exit 1
fi

if grep -R -E -- '--agent claude.*--effort medium|--model (sonnet|opus|haiku).*--effort medium|Claude defaults to `medium`|Claude reasoning effort defaults to `medium`|accepts `medium` or `high`' README.md docs skills; then
  echo "stale Claude medium effort example found"
  exit 1
fi

if grep -R -F -- '--model pro' README.md docs skills; then
  echo "unsupported Gemini model example found"
  exit 1
fi

step "shim smoke"
set +e
status_output=$(bin/alt-subagent --status missing-job 2>&1)
status_code=$?
set -e

if [ "$status_code" -ne 4 ]; then
  echo "expected missing-job smoke to exit 4, got $status_code"
  echo "$status_output"
  exit 1
fi

printf '%s\n' "$status_output" | grep -q '"status":"failed"'
printf '%s\n' "$status_output" | grep -q '"job_id":"missing-job"'
printf '%s\n' "$status_output" | grep -q '"exit_code":4'

step "validation complete"
