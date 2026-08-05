---
name: ci-debug
description: >
  Reports CI status on a LoteoApp/LoteosAPP pull request — PR, head commit,
  and pass/fail per check — and if anything failed, diagnoses the actual
  root cause by reproducing the job locally instead of fetching Actions logs
  (which are blocked in Cowork's sandbox — see below). Use this whenever the
  user reports a CI check failed, asks why a PR's checks are red, asks about
  the status of a PR's checks (even if nothing is known to have failed yet),
  pastes a GitHub Actions error, or asks to look into a failing
  build/test/coverage job on this repo. Always report the PR/commit and the
  full pass/fail list first, even when everything is green — only go into
  root-cause diagnosis for checks that actually failed. If invoked without a
  specific PR/branch/commit, resolve it from the open PR on the current
  branch and ask the user if there isn't exactly one — never assume "the
  last PR mentioned in conversation". Don't waste time retrying the GitHub
  logs API against Azure Blob Storage URLs — go straight to local
  reproduction.
compatibility: Requires a GitHub personal access token with repo scope, and
  the frontend toolchain (pnpm/node) available in the sandbox. Backend (Go)
  and Docker checks cannot be reproduced locally — see step 4.
---

# LoteosAPP: diagnose a failing CI check

## 1. Figure out which PR you're checking

If the user named a PR number, branch, or commit, use that. Otherwise, don't
guess or default to "the last PR I opened" — resolve it from the current
state of the repo:

```bash
BRANCH=$(git -C <repo-root> branch --show-current)
curl -s -H "Authorization: token $TOKEN" \
  "https://api.github.com/repos/LoteoApp/LoteosAPP/pulls?head=LoteoApp:$BRANCH&state=open"
```

If that returns exactly one open PR, use its `head.sha` and move on. If it
returns none, ask the user which PR or branch to check instead of picking
one arbitrarily — there may be several open PRs and no way to know which one
they mean. If somehow more than one comes back (shouldn't normally happen
for a single branch), also ask rather than picking the first.

## 2. Report the PR status first, always

Before deciding whether there's anything to diagnose, always tell the user
which PR this is about: number, title, and the head commit (short SHA is
fine — from the PR object's `head.sha`). Then pull the check runs and report
them as a simple pass/fail list — this is the baseline report regardless of
whether anything actually failed:

```bash
curl -s -H "Authorization: token $TOKEN" \
  "https://api.github.com/repos/LoteoApp/LoteosAPP/commits/<head_sha>/check-runs" \
  | python3 -c "import json,sys; d=json.load(sys.stdin); [print(c['name'], c['conclusion'], c['html_url']) for c in d['check_runs']]"
```

If every check succeeded, say so plainly (PR #, commit, "todo verde") and
stop — don't invent a diagnosis for something that isn't failing, and don't
silently skip reporting the good state either. Only continue to step 3 if
at least one check's conclusion is `failure` (or still `in_progress` / has
no conclusion yet, which is worth flagging too — say it's still running
rather than treating it as a failure).

## 3. Try annotations before logs

```bash
curl -s -H "Authorization: token $TOKEN" \
  "https://api.github.com/repos/LoteoApp/LoteosAPP/check-runs/<job_id>/annotations"
```

This is fast and sometimes enough (compiler errors, lint messages). For
generic failures it often just says "Process completed with exit code 1" —
that's not a dead end, it's the signal to move to step 4.

**Don't bother with `GET /actions/jobs/{id}/logs`.** It 302-redirects to
`productionresultssa16.blob.core.windows.net` (or similar), which Cowork's
sandbox proxy blocks (`403 blocked-by-allowlist`) — confirmed, not worth
retrying or trying `-L` variations. `mcp__workspace__web_fetch` on the
`html_url` also won't help: the Actions log viewer is behind login and
client-rendered.

## 4. Reproduce locally

Run `scripts/reproduce-ci.sh <repo-root>`. It syncs the frontend to a scratch
copy and runs the same commands as the `build`, `test`, `coverage`, and
`dependency-audit` (frontend half) jobs in `.github/workflows/ci.yml`, so
whatever fails, fails the same way here — with full output, not a one-line
annotation.

If the failing job is backend-only (`go vet`/`go test`/`govulncheck`) or
`compose-config`, the script can't run it (no Go toolchain, no Docker in the
sandbox). Say this explicitly and ask the user to run the equivalent command
locally — don't guess at what the error might be from the annotation alone.

## 5. Explain the diagnosis before touching anything

Report back to the user in plain terms: which job failed, the actual root
cause (not just "exit code 1" — point at the specific line/threshold/error
from the reproduction output), and why it happened. This is the actual point
of this skill — reproducing the failure is a means to that explanation, not
the deliverable by itself. Only after that should you propose a fix; if the
fix is non-obvious or touches more than the failing check, say what you plan
to do and let the user weigh in before changing anything.

## 6. Fix and hand off

Once the user's on board with the diagnosis, fix it in the working tree and
use the `open-pr` skill to verify, show the diff, and push the fix — don't re-run
`reproduce-ci.sh` as a substitute for that skill's `verify.sh`; they overlap
on purpose (same underlying checks) but `open-pr` is the one that also handles
docs review and the PR body.

## Known root causes worth checking first

These have come up before and are quick to rule out:

- **Per-file coverage thresholds**: `apps/frontend/vite.config.ts` sets
  `perFile: true` with 80% lines/functions/statements and 75% branches. Any
  new `.tsx`/`.ts` under `src/features/**` with zero tests fails coverage
  immediately, even a trivial placeholder component. Fix: add a render test,
  don't touch the threshold.
- **`test:backend:coverage` path drift**: the root `package.json` script
  points at explicit backend package paths
  (`./internal/business/usecase ./internal/infrastructure/delivery/webapp/handler`).
  If the backend gets reorganized again, this script silently breaks. It's
  intentionally explicit paths rather than `./...` — keep it that way unless
  the team decides otherwise.
- **Go stdlib CVEs via govulncheck**: caused by a bare `go 1.x.0` in
  `apps/backend/go.mod` resolving to an unpatched initial release. Fix by
  bumping to the latest patch version of that same minor (e.g. `1.26.0` →
  `1.26.5`), not by upgrading the minor version.
