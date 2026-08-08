---
name: open-pr
description: >
  End-to-end workflow for pushing a change and opening a pull request in the
  LoteoApp/LoteosAPP repo (branch naming, commit message format, local
  verification, PR body template, and the GitHub API calls to push and open
  the PR). Use this whenever the user asks to push changes, commit and push,
  open/create a PR, or wraps up a branch of work in this repo — even if they
  just say "pushea esto" or "armemos el PR" without spelling out the steps.
  Also use it as a checklist before pushing any change to this repo, since it
  encodes house rules (English commit/PR titles, Spanish PR body, docs review,
  explicit push confirmation) that are easy to forget mid-task.
compatibility: Requires a GitHub personal access token with repo scope (ask
  the user if one hasn't been shared in the session) and network access to
  api.github.com and github.com.
---

# LoteosAPP: verify, push, and open a PR

This encodes the repo's actual workflow (see `AGENTS.md`, which is the source
of truth — skim it if anything here looks stale) plus sandbox-specific
workarounds discovered while working in Cowork. Follow it in order.

## 1. Branch

Always branch from a freshly pulled `develop` (never from a stale local copy —
other PRs merge frequently):

```bash
git checkout develop
git pull <remote-with-token> develop
git checkout -b <type>/<issue-number>-<slug>
```

`<type>` is one of `feat`, `fix`, `refactor`, `test`, `docs`, `chore` — same
list as commit types, see below. `<issue-number>` ties the branch to a GitHub
issue when one exists (e.g. a task created by the `create-tasks`
skill); omit it if there's no issue. Example: `feat/79-configure-router`.

The one exception to "branch from develop": a hotfix destined for `main`
directly uses `hotfix/<slug>` (see the `main-source-branch` CI check, which
only allows PRs into `main` from `develop` or `hotfix/*`).

## 2. Commit message

- Simplified Conventional Commits: `<type>: <description>`.
- Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`.
- Description in **English**, imperative, lowercase, no trailing period.
- Examples: `feat: add authentication`, `fix: correct installment
  calculation`, `docs: document customer endpoint`.

This is a common slip: the PR *title* must follow the same rule (English,
same format) even though the PR *body* is in Spanish (see step 4). Don't mix
languages within the title.

## 3. Verify locally before showing the diff

Run `scripts/verify.sh <repo-root>` (defaults to cwd). It mirrors the
frontend CI jobs (`build`, `test`, `coverage`) against a scratch copy under
`/tmp`, because the mounted repo folder in Cowork typically has no
`node_modules` installed and writing into the FUSE mount directly is
unreliable. Read the script before running it once so you know what it's
doing; it's short.

If the diff touches `apps/backend`, the script cannot check it (no Go
toolchain in the sandbox) — say so explicitly to the user rather than
silently skipping, and suggest they run `go vet ./...` / `go test ./...` /
`go build ./...` from `apps/backend` locally, or rely on CI plus the
`ci-debug` skill after pushing.

If verification fails, fix the root cause before proceeding — don't push
red. A common one: per-file coverage thresholds (`vite.config.ts`,
`perFile: true`, 80% lines/functions/statements, 75% branches) reject any new
`.tsx`/`.ts` file under `src/features/**` that has zero test coverage, even a
tiny placeholder component. Add a minimal render test rather than lowering
the threshold.

## 4. Documentation review

Before pushing, check whether the change leaves `README.md` or anything
under `docs/` stale (this is an explicit AGENTS.md rule, easy to miss because
nothing enforces it automatically). Ask yourself: does this change what a
new contributor would read in those docs to understand the system? If yes,
update the doc in the same PR. If genuinely nothing applies, that's fine —
just don't skip the check.

## 5. Show the diff and get confirmation

**Never push without showing the user what's about to go out and getting an
explicit yes.** Show: the branch name, the commit(s) (`git log
develop..HEAD --oneline`), and `git status --short` / `git diff --stat` for
anything uncommitted. Wait for confirmation before the next step.

## 6. Push

```bash
git push https://<username>:<token>@github.com/LoteoApp/LoteosAPP.git <branch>:<branch>
```

If no token is in the session's environment or recent context, ask the user
for one — don't guess or reuse a token from an unrelated context. Never print
the token back in full in your response to the user.

## 7. Open the PR

Use the GitHub REST API (`POST /repos/LoteoApp/LoteosAPP/pulls`). Base branch
is `develop` unless this is a hotfix (base `main`). Title follows the same
Conventional Commit + English rule as commits. Body **must** use this exact
template (Spanish, from AGENTS.md) — don't paraphrase the headings:

```
## Qué se hizo

Descripción breve del cambio.

## Cómo probarlo

Pasos para levantar el entorno y probar el cambio.

## Cambios en base de datos

Migraciones agregadas o modificadas, si aplica.

## Capturas

Agregar capturas si hay cambios visuales.
```

Link the PR to its task automatically, don't wait to be told: parse the
issue number out of the branch name (the `<issue-number>` in
`<type>/<issue-number>-<slug>` from step 1, e.g. `87` in
`feat/87-secret-scanning`). If the branch name has one, add `Tarea: #<number>`
at the end of the PR body — a plain reference, not a closing keyword.

Don't use `Closes`/`Fixes`/`Resolves`: those only auto-close the issue when
the PR merges into the repo's *default* branch, which on GitHub is `main`
for this repo. Every PR here merges into `develop`, never directly into
`main`, so a closing keyword would silently never fire — it looks like it
should close the task but doesn't, which is worse than not claiming it at
all. Closing the task, if it should happen when this PR merges, is a
separate manual step (or a future explicit `PATCH` to the issue), not
something the PR body can reliably trigger in this repo's branching model.

If the branch name doesn't follow the `<type>/<issue-number>-<slug>`
pattern (no leading number after the type, e.g. `chore/agent-skills`), that
means this branch was never tied to a task — don't add a `Tarea:` line, and
don't ask the user for an issue number or treat it as missing information.
Not every PR has a task behind it, and that's a normal, expected case, not
an error.

If the branch type is `docs`, label the PR `documentation` right after
creating it — don't wait to be asked:

```bash
curl -s -X POST -H "Authorization: token $TOKEN" -H "Accept: application/vnd.github+json" \
  "https://api.github.com/repos/LoteoApp/LoteosAPP/issues/<pr_number>/labels" \
  -d '{"labels": ["documentation"]}'
```

(PRs use the same labels endpoint as issues.) Other branch types don't have
an established label mapping yet — don't guess one.

Report the resulting PR URL to the user.
