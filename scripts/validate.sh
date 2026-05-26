#!/usr/bin/env sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT"

step() {
  printf '\n==> %s\n' "$1"
}

step "go tests"
go test ./...

step "build"
scripts/build.sh
test -x dist/codex-implement
test -x bin/codex-implement

step "plugin metadata"
grep -q '"name": "codex-implement"' .claude-plugin/plugin.json
grep -q 'name: codex-implement' skills/codex-implement/SKILL.md
grep -q 'allowed-tools: Bash' skills/codex-implement/SKILL.md

step "documentation examples"
grep -q 'make build' README.md
grep -q -- '--effort high' README.md
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

if grep -R -F -- '--effort xhigh' README.md docs skills; then
  echo "unsupported xhigh effort example found"
  exit 1
fi

step "shim smoke"
set +e
status_output=$(bin/codex-implement --status missing-job 2>&1)
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
