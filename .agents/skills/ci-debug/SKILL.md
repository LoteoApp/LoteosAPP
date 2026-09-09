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
  specific PR/branch/commit, don't resolve an associated PR from GitHub —
  there may not be one open yet. Instead run the full CI suite locally
  against the current branch (uncommitted changes included), simulating
  what GitHub Actions would run, and report the pass/fail summary directly.
  This is the mode to use right before pushing, to catch failures before
  they hit CI — never assume "the last PR mentioned in conversation" as a
  substitute for either mode. Don't waste time retrying the GitHub logs API
  against Azure Blob Storage URLs — go straight to local reproduction.
compatibility: For PR diagnosis, requires a GitHub personal access token
  with repo scope. For either mode, requires the frontend toolchain
  (pnpm/node) available in the sandbox. Backend (Go) and Docker checks are
  only reproduced locally if go/govulncheck/docker are actually installed
  in the sandbox — see step 4.
---

# LoteosAPP: diagnose a failing CI check

## 1. Decide: PR diagnosis, or local pre-push run?

If the user named a PR number, branch, or commit to check against GitHub,
use the **PR diagnosis** flow: resolve it and continue with step 2.

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

If the user did **not** name a PR, branch, or commit — e.g. "revisá el CI
antes de pushear", "¿el CI va a pasar?", or any ask to check CI status with
nothing more specific to go on — don't try to resolve an associated PR from
GitHub either. There may not be one open yet, and this is almost always
someone about to push who wants to catch failures before they happen. Skip
the GitHub API entirely and go straight to the **local pre-push run**: run
`scripts/reproduce-ci.sh <repo-root>` against the current branch's working
tree (uncommitted changes included) and report the pass/fail summary it
prints, per step 4. Steps 2 and 3 don't apply in this mode — there's no PR
or check-run to query yet.

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

Run `scripts/reproduce-ci.sh <repo-root>`. It syncs the repo to a scratch
copy and runs the same commands as every job in `.github/workflows/ci.yml`
except `dependency-review` (which needs a real PR diff and only runs on
GitHub): `build`, `test`, `coverage`, and `dependency-audit` for frontend
*and* backend, plus `compose-config`. Whatever fails, fails the same way
here — with full output, not a one-line annotation — and it ends with a
PASS/FAIL summary (also reflected in its exit code).

Backend (`go vet`/`go build`/`go test`/`govulncheck`) and `compose-config`
only run if `go`/`govulncheck`/`docker` are actually available in the
sandbox; the script checks for each and skips with an explicit message
rather than failing silently or guessing. If something got skipped and the
change touches `apps/backend` or `compose.yaml`, say so and ask the user to
run the equivalent command locally.

In **local pre-push run** mode, this step is the whole skill: run the
script, show the summary, and stop — there's no PR to report status for and
nothing has failed on GitHub yet, just tell the user what would happen if
they pushed now. In **PR diagnosis** mode, use this to reproduce the
specific job GitHub reported as failing (matching the name from step 2's
list) before moving to step 5.

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

In **local pre-push run** mode there's no diagnosis hand-off needed if
everything passed — just report the summary and let the user decide when to
push (`open-pr` skill from there). If something failed, walk through step 5
first before touching anything, same as PR diagnosis mode.

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
