#!/usr/bin/env bash
# Reproduces the frontend-side CI jobs from .github/workflows/ci.yml locally,
# so a failing check can be diagnosed without fetching Actions logs (which
# are hosted on Azure Blob Storage and blocked by Cowork's sandbox network
# allowlist — see SKILL.md for how that was confirmed).
#
# Covers: build (typecheck+build), test, coverage, dependency-audit's
# frontend half (pnpm audit). Does NOT cover: backend go vet/build/test/
# govulncheck (no Go toolchain in the sandbox), compose-config (no Docker),
# dependency-review (GitHub-hosted action, not locally runnable). Say so to
# the user for those instead of silently skipping.
#
# Usage: scripts/reproduce-ci.sh [path-to-repo-root]

set -euo pipefail

REPO_ROOT="${1:-$(pwd)}"
SCRATCH_DIR="${LOTEOSAPP_SCRATCH_DIR:-/tmp/loteosapp-verify}"

echo "== Syncing $REPO_ROOT -> $SCRATCH_DIR (excluding node_modules, .git) =="
mkdir -p "$SCRATCH_DIR"
rsync -a --delete --exclude 'node_modules' --exclude '.git' --exclude 'apps/backend' \
  "$REPO_ROOT"/ "$SCRATCH_DIR"/

cd "$SCRATCH_DIR"

if [ ! -d node_modules ] || [ package.json -nt node_modules ] || [ pnpm-lock.yaml -nt node_modules ]; then
  echo "== pnpm install --frozen-lockfile =="
  corepack pnpm install --frozen-lockfile
  touch node_modules
fi

run_job() {
  local label="$1"; shift
  echo
  echo "== $label =="
  if "$@"; then
    echo "-- $label: PASSED --"
  else
    echo "-- $label: FAILED (see output above) --"
  fi
}

run_job "build / frontend typecheck" corepack pnpm --filter @loteos/frontend typecheck
run_job "build / frontend build" corepack pnpm --filter @loteos/frontend build
run_job "test / frontend" corepack pnpm --filter @loteos/frontend test
run_job "coverage / frontend" corepack pnpm --filter @loteos/frontend test:coverage
run_job "dependency-audit / frontend (pnpm audit)" corepack pnpm audit --audit-level=high

cat <<'EOF'

== Not reproducible in this sandbox ==
- build / test / coverage / dependency-audit for apps/backend: no Go
  toolchain here. Ask the user to run `go vet ./...`, `go build ./...`,
  `go test ./...`, and `govulncheck ./...` from apps/backend locally.
- compose-config: no Docker here. Ask the user to run
  `docker compose config` locally.
- dependency-review: runs a GitHub-hosted action against the PR diff; there's
  no local equivalent, read the check's annotations instead (see SKILL.md).
EOF
