#!/usr/bin/env bash
# Runs the same checks CI runs (build, test, coverage) against a scratch copy
# of the repo, so failures show up before push instead of after.
#
# Why a scratch copy: on Cowork's sandbox, the mounted repo folder
# (D:\LoteosApp) is a FUSE mount that does not have node_modules installed
# and can be flaky for pnpm/go to write into directly. Copying to a local
# /tmp directory once per session and installing there is faster and more
# reliable than fighting the mount.
#
# Usage: scripts/verify.sh [path-to-repo-root]
#   path-to-repo-root defaults to the current directory.

set -euo pipefail

REPO_ROOT="${1:-$(pwd)}"
SCRATCH_DIR="${LOTEOSAPP_SCRATCH_DIR:-/tmp/loteosapp-verify}"

echo "== Syncing $REPO_ROOT -> $SCRATCH_DIR (excluding node_modules, .git) =="
mkdir -p "$SCRATCH_DIR"
rsync -a --delete --exclude 'node_modules' --exclude '.git' --exclude 'apps/backend' \
  "$REPO_ROOT"/ "$SCRATCH_DIR"/

cd "$SCRATCH_DIR"

echo "== Installing frontend dependencies (only if lockfile changed) =="
if [ ! -d node_modules ] || [ package.json -nt node_modules ] || [ pnpm-lock.yaml -nt node_modules ]; then
  corepack pnpm install --frozen-lockfile
  touch node_modules
else
  echo "node_modules looks up to date, skipping install."
fi

echo "== Frontend typecheck =="
corepack pnpm --filter @loteos/frontend typecheck

echo "== Frontend build =="
corepack pnpm --filter @loteos/frontend build

echo "== Frontend test + coverage =="
corepack pnpm --filter @loteos/frontend test:coverage

cat <<'EOF'

== Backend (Go) checks were NOT run ==
The sandbox has no Go toolchain, so `go vet ./...`, `go build ./...`,
`go test ./...` and `govulncheck ./...` from apps/backend cannot run here.
If the PR touches apps/backend, ask the user to run those locally (they have
Go set up), or wait for the CI `build`, `test`, `coverage` and
`dependency-audit` jobs and check the results with the ci-debug
skill once the PR is pushed.
EOF
