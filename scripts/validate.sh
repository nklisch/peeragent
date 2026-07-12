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
test -x dist/peeragent
test -x bin/peeragent

step "plugin package"
scripts/package-plugin.sh
test -x plugin/bin/peeragent
test -f plugin/.claude-plugin/plugin.json
test -f plugin/.codex-plugin/plugin.json
test -f plugin/skills/peer/SKILL.md
test -f plugin/skills/peer-review/SKILL.md

# Curated-tree sync enforcement lives in CI (.github/workflows/build-binaries.yml),
# which runs on committed state. It is intentionally NOT here: validate.sh runs
# mid-bump (scripts/bump.sh regenerates plugin/ for the new version before
# committing), where uncommitted plugin/ changes are expected and legitimate.

step "committed platform binaries"
for t in linux-amd64 linux-arm64 darwin-amd64 darwin-arm64; do
  test -x "plugin/bin/$t/peeragent"
  test -s "plugin/bin/$t/peeragent"
done

set +e
committed_out=$(plugin/bin/peeragent --status missing-job 2>&1)
committed_code=$?
set -e
if [ "$committed_code" -ne 4 ]; then
  echo "committed-binary smoke expected exit 4, got $committed_code"
  echo "$committed_out"
  exit 1
fi
printf '%s\n' "$committed_out" | grep -q '"status":"failed"'
printf '%s\n' "$committed_out" | grep -q '"exit_code":4'

set +e
notinstalled_out=$(PEERAGENT_TARGET_OVERRIDE=plan9-sparc plugin/bin/peeragent --agent codex x 2>&1)
notinstalled_code=$?
set -e
if [ "$notinstalled_code" -ne 3 ]; then
  echo "not-installed smoke expected exit 3, got $notinstalled_code"
  echo "$notinstalled_out"
  exit 1
fi
printf '%s\n' "$notinstalled_out" | grep -q '"exit_code":3'
printf '%s\n' "$notinstalled_out" | grep -q 'releases'

step "release artifacts"
scripts/release.sh "$VERSION"
test -f "dist/release/peeragent_${VERSION}_linux_amd64.tar.gz"
test -f "dist/release/peeragent_${VERSION}_linux_arm64.tar.gz"
test -f "dist/release/peeragent_${VERSION}_darwin_amd64.tar.gz"
test -f "dist/release/peeragent_${VERSION}_darwin_arm64.tar.gz"
test -f dist/release/checksums.txt

step "plugin metadata"
grep -q '"name": "peeragent"' .claude-plugin/plugin.json
grep -q '"name": "peeragent"' .codex-plugin/plugin.json
grep -q '"name": "@nklisch/pi-peeragent"' package.json
grep -q '"version": "'"$VERSION"'"' package.json
grep -q '"./plugin/skills"' package.json
grep -q '"name": "peeragent"' .claude-plugin/marketplace.json
grep -q '"name": "peeragent"' .agents/plugins/marketplace.json
grep -q './plugin' .claude-plugin/marketplace.json
grep -q './plugin' .agents/plugins/marketplace.json
grep -q '"skills": "./skills/"' .codex-plugin/plugin.json
grep -q '"defaultPrompt"' .codex-plugin/plugin.json
grep -q 'name: peer' skills/peer/SKILL.md
grep -q 'allowed-tools: Bash' skills/peer/SKILL.md
grep -q 'name: peer-review' skills/peer-review/SKILL.md

step "skill metadata constraints"
for skill in skills/*/SKILL.md plugin/skills/*/SKILL.md; do
  [ -f "$skill" ] || continue
  description_length=$(awk '
    NR == 1 && $0 == "---" { in_frontmatter = 1; next }
    in_frontmatter && $0 == "---" { in_frontmatter = 0; in_description = 0; next }
    in_frontmatter && /^description:[[:space:]]*>?[[:space:]]*$/ {
      in_description = 1
      next
    }
    in_frontmatter && /^description:[[:space:]]*/ {
      line = $0
      sub(/^description:[[:space:]]*/, "", line)
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", line)
      total += length(line)
      in_description = 0
      next
    }
    in_frontmatter && in_description && /^[[:space:]]/ {
      line = $0
      sub(/^[[:space:]]+/, "", line)
      total += length(line) + 1
      next
    }
    in_frontmatter && in_description { in_description = 0 }
    END { print total }
  ' "$skill")
  if [ "$description_length" -gt 1024 ]; then
    echo "$skill description exceeds 1024 characters ($description_length)" >&2
    exit 1
  fi
done

step "documentation examples"
grep -q 'make build' README.md
grep -q 'claude plugin marketplace add nklisch/peeragent' README.md
grep -q 'codex plugin marketplace add nklisch/peeragent' README.md
grep -q "pi install git:github.com/nklisch/peeragent@v$VERSION" README.md
grep -q "make release VERSION=$VERSION" README.md
grep -q -- '--agent gemini' README.md
grep -q -- '--agent claude' README.md
grep -q -- '--agent zai' README.md
grep -q -- '--effort high' README.md
grep -q -- '--effort xhigh' README.md
grep -q -- '--model luna' README.md
grep -q -- '--model terra' README.md
grep -q -- '--model sol' README.md
grep -q -- '--model fable' README.md
grep -q -- '--model opus' README.md
grep -q -- '--model gemini-3.5' README.md
grep -q -- '--model glm-5.2' README.md
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

if grep -R -E -- '--agent (claude|zai)[^|`]*--effort low|--model (fable|sonnet|opus|haiku|glm-5\.2)[^|`]*--effort low' README.md docs skills; then
  echo "low effort is supported only for Codex"
  exit 1
fi

if grep -R -E -- '--agent claude[^|`]*--effort medium|--model (fable|sonnet|opus|haiku)[^|`]*--effort medium|Claude defaults to `medium`|Claude reasoning effort defaults to `medium`|accepts `medium` or `high`' README.md docs skills; then
  echo "stale Claude medium effort example found"
  exit 1
fi

if grep -R -F -- '--ask-for-approval' README.md docs skills cmd internal; then
  echo "unsupported Codex --ask-for-approval flag found"
  exit 1
fi

if grep -R -F -- '--model pro' README.md docs skills; then
  echo "unsupported Gemini model example found"
  exit 1
fi

if grep -R -E -- 'glm-(4|5\.1|5-turbo|5v)' README.md docs skills; then
  echo "unsupported Z.AI model example found"
  exit 1
fi

step "shim smoke"
set +e
status_output=$(bin/peeragent --status missing-job 2>&1)
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

step "shim PEERAGENT_BIN self-exec-loop guard"
# PEERAGENT_BIN must point at the Go BINARY, not a shim copy. Pointing it at a
# shim makes step 1 `exec` the shim again with the same env var -> infinite
# self-exec loop that burns a core and never runs the agent. The guard detects
# a shebang (shim) target and fails fast (exit 2) instead of spinning.
# Run under `timeout` so a regression (guard removed/broken) hangs the test
# rather than passing by luck.
set +e
loop_output=$(timeout 3 env PEERAGENT_BIN="$ROOT/bin/peeragent" bin/peeragent --status missing-job 2>&1)
loop_code=$?
set -e
if [ "$loop_code" -ne 2 ]; then
  echo "expected PEERAGENT_BIN->shim to exit 2 (guard), got $loop_code (124=spun/looped=REGRESSION)"
  echo "$loop_output"
  exit 1
fi
printf '%s\n' "$loop_output" | grep -q '"exit_code":2'
printf '%s\n' "$loop_output" | grep -q 'self-exec loop'

step "validation complete"
