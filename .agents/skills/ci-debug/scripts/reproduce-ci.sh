#!/usr/bin/env bash
# Reproduces the CI jobs from .github/workflows/ci.yml locally, so a failing
# check can be diagnosed -- or caught before push -- without fetching
# Actions logs, which are hosted on Azure Blob Storage and blocked by
# Cowork's sandbox network allowlist (see SKILL.md for how that was
# confirmed).
#
# Covers: build (frontend typecheck+build, backend vet+build), test
# (frontend + backend), coverage (frontend + backend), dependency-audit
# (pnpm audit + govulncheck), compose-config (docker compose config).
# Backend/govulncheck/docker steps only run if the corresponding toolchain
# (go, govulncheck, docker) is actually present -- skipped explicitly
# otherwise, never guessed at. dependency-review is NOT covered: it runs
# against a PR diff on a GitHub-hosted action and has no local equivalent.
#
# Usage: scripts/reproduce-ci.sh [path-to-repo-root]
#   path-to-repo-root defaults to the current directory.
#
# Exit code reflects overall pass/fail (non-zero if anything failed).

set -euo pipefail

REPO_ROOT="${1:-$(pwd)}"
SCRATCH_DIR="${LOTEOSAPP_SCRATCH_DIR:-/tmp/loteosapp-verify}"

echo "== Syncing $REPO_ROOT -> $SCRATCH_DIR (excluding node_modules, .git) =="
mkdir -p "$SCRATCH_DIR"
rsync -a --delete --exclude 'node_modules' --exclude '.git' \
  "$REPO_ROOT"/ "$SCRATCH_DIR"/

cd "$SCRATCH_DIR"

if [ ! -d node_modules ] || [ package.json -nt node_modules ] || [ pnpm-lock.yaml -nt node_modules ]; then
  echo "== pnpm install --frozen-lockfile =="
  corepack pnpm install --frozen-lockfile
  touch node_modules
fi

PASS=()
FAIL=()

run_job() {
  local label="$1"; shift
  echo
  echo "== $label =="
  if "$@"; then
    echo "-- $label: PASSED --"
    PASS+=("$label")
  else
    echo "-- $label: FAILED (see output above) --"
    FAIL+=("$label")
  fi
}

skip_job() {
  local label="$1" reason="$2"
  echo
  echo "== $label: SKIPPED -- $reason =="
}

run_job "build / frontend typecheck" corepack pnpm --filter @loteos/frontend typecheck
run_job "build / frontend build" corepack pnpm --filter @loteos/frontend build
run_job "test / frontend" corepack pnpm --filter @loteos/frontend test
run_job "coverage / frontend" corepack pnpm --filter @loteos/frontend test:coverage
run_job "dependency-audit / frontend (pnpm audit)" corepack pnpm audit --audit-level=high

if command -v go >/dev/null 2>&1; then
  run_job "build / backend vet"   bash -c 'cd apps/backend && go vet ./...'
  run_job "build / backend build" bash -c 'cd apps/backend && go build ./...'
  run_job "test / backend"        bash -c 'cd apps/backend && go test ./...'
  run_job "coverage / backend"    bash -c 'cd apps/backend && go test -count=1 -coverprofile coverage.out ./internal/business/usecase ./internal/infrastructure/delivery/webapp/handler && go tool cover -func coverage.out'

  if command -v govulncheck >/dev/null 2>&1; then
    run_job "dependency-audit / backend (govulncheck)" bash -c 'cd apps/backend && govulncheck ./...'
  else
    skip_job "dependency-audit / backend (govulncheck)" "govulncheck not installed here (go install golang.org/x/vuln/cmd/govulncheck@latest)"
  fi
else
  skip_job "build/test/coverage/dependency-audit (backend)" "no Go toolchain in this sandbox -- ask the user to run go vet/build/test and govulncheck from apps/backend locally"
fi

if command -v docker >/dev/null 2>&1; then
  run_job "compose-config" docker compose config
else
  skip_job "compose-config" "no Docker in this sandbox -- ask the user to run 'docker compose config' locally"
fi

echo
echo "== dependency-review: NOT REPRODUCIBLE -- runs against the PR diff on a GitHub-hosted action, no local equivalent =="

echo
echo "======================================"
echo "SUMMARY: ${#PASS[@]} passed, ${#FAIL[@]} failed"
for j in "${PASS[@]:-}"; do [ -n "$j" ] && echo "  PASS  $j"; done
for j in "${FAIL[@]:-}"; do [ -n "$j" ] && echo "  FAIL  $j"; done
echo "======================================"

if [ "${#FAIL[@]}" -gt 0 ]; then
  exit 1
fi
