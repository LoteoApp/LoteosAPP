---
name: pr-review
description: >
  Reviews a pull request in the LoteoApp/LoteosAPP repo (or the local diff
  before opening one) against AGENTS.md and docs/: naming, architecture and
  layer boundaries, error handling, test coverage and quality, migrations,
  PR title/body format, dependency justification, and whether the change
  matches what its own description and the project docs say it should do.
  Drafts inline line comments plus a summary and always shows the draft in
  chat first — only posts to GitHub, as a non-blocking COMMENT review (never
  APPROVE/REQUEST_CHANGES), after explicit confirmation. Use whenever the
  user asks to review a LoteosAPP PR, wants a code review before pushing,
  mentions "revisá este PR", "hacé code review", "dale una mirada al diff
  antes de pushear", or asks what's wrong with a PR/branch relative to the
  repo's conventions.
compatibility: PR-mode requires a GitHub personal access token with repo
  scope (same as open-pr / ci-debug). Local-diff mode needs no token.
---

# LoteosAPP: review a pull request against the repo's own conventions

This is for the LoteoApp/LoteosAPP GitHub repo. `AGENTS.md` at the repo root is the source of truth for conventions — skim it fresh each time in case it changed, don't rely on a cached memory of it. The repo is checked out locally; adapt paths to wherever it's mounted in the current session.

This skill is specific to LoteosAPP's own rules (layering, naming, coverage thresholds, PR template, docs). It is not a general security audit — if the user wants a deep security pass, that's a separate concern; this skill only catches the security basics that fall out of the architecture rules (e.g. business logic depending on raw SQL, missing authz checks visible in a handler diff).

## 1. Decide the mode

- **User named a PR** (number, URL, or "el PR de X"): review it via the GitHub API against its actual diff. Continue at step 2.
- **User didn't name a PR** ("revisá mi rama antes de pushear", "dale una mirada a esto antes de abrir el PR", or just asked for a review with nothing more specific): there may be no PR yet. Review the local diff of the current branch against `develop` instead — skip straight to step 3, there's nothing to fetch from GitHub. This mirrors how `ci-debug` picks its mode.

If the user's branch looks like `<type>/<issue>-<slug>`, note the issue number — useful context for whether the change matches what was asked for, but don't go fetch the issue unless the diff leaves you unsure what the PR is trying to do.

## 2. Fetch the PR diff (PR mode only)

```bash
curl -s -H "Authorization: token $TOKEN" \
  "https://api.github.com/repos/LoteoApp/LoteosAPP/pulls/<pr_number>/files?per_page=100"
```

Each entry has `filename`, `status`, `patch` (the unified diff hunk — this is what you need for line-level comments), and counts. If `patch` is missing for a file (binary, or too large), note that you couldn't review it line-by-line and say so rather than guessing at its contents.

Also fetch the PR body and title (`GET /repos/LoteoApp/LoteosAPP/pulls/<pr_number>`) — you need these to check the PR template and Conventional Commit title, and the body's "Qué se hizo" tells you what the change is supposed to accomplish, which is what you'll check the actual diff against.

## 3. Fetch the local diff (local mode only)

```bash
git -C <repo-root> fetch origin develop
git -C <repo-root> diff origin/develop...HEAD
```

Include uncommitted changes too (`git status --short` / `git diff`) — the point of local mode is catching things before they're even committed, so don't ignore working-tree changes.

## 4. Load context before judging anything

Read (or re-skim) these before forming opinions on the diff — the whole point of this skill is catching deviations from what the repo already says it wants, not generic code review:

- `AGENTS.md` — package manager, layout, architecture, testing/coverage thresholds, commit format, PR template.
- `docs/architecture.md` — layer boundaries and dependency direction in detail.
- Any other file under `docs/` that the diff plausibly touches the subject matter of (e.g. `docs/database.md` for migrations, `docs/domain.md` for business rules) — you don't need to read all of them every time, just the ones relevant to what changed.
- A file or two of existing code in the same area the diff touches, if you're unsure what the established pattern looks like (e.g. how existing use cases are structured, how existing frontend features are organized). Don't invent a "correct" pattern from general Go/React knowledge when the repo already has one.

## 5. Review checklist

Go through the diff with these in mind. Not everything applies to every PR — a docs-only PR has no coverage to check, a migration-only PR has no frontend naming to check. Use judgment about what's relevant instead of forcing every item onto every file.

**Architecture and layering**
- Backend: `internal/business` (domain, gateway, usecase) must not import HTTP or concrete `pgx` types; use cases depend on `gateway` contracts, never a concrete adapter directly.
- Adapters (persistence, config, HTTP delivery/server) stay under `internal/infrastructure`; DI wiring stays in `dependencies` and only builds objects.
- Frontend: composition in `src/app`, feature code in `src/features`, reusable code in `src/shared`. A feature must not import another feature's internal files.
- No new `utils`, `helpers`, `common`, `controllers`, `services`, `repositories`, or `models` grab-bag directories. No new abstraction or dependency unless the diff shows a concrete use case for it right now.

**Naming**
- Matches the vocabulary already used in the surrounding package/feature (domain terms from `docs/domain.md`, not ad hoc synonyms).
- Files, types, and functions follow the naming already established nearby — new code shouldn't introduce a second convention alongside an existing one.

**Error handling**
- Errors are wrapped/propagated consistently with how nearby code already does it (don't accept a silent swallow next to code that otherwise always wraps and logs).
- HTTP handlers map errors to sensible status codes; look for the obvious gaps (a validation failure returning 500, a not-found returning 200 with empty body, etc.).

**Tests and coverage**
- Every new behavior and bug fix has a test in the same diff — this is a hard AGENTS.md rule, not a suggestion.
- Backend: business services tested with small fakes of their interfaces (not heavy mocks), handlers tested through request/response, repositories only need integration tests (won't run in this sandbox — note that rather than skip the check).
- Frontend: tests query/assert the UI the way a user would (Testing Library), not internal state, Tailwind classes, or private functions.
- Coverage: changed code should clearly be at or above 80% lines/statements/functions and 75% branches (90% for anything that looks like a critical business rule). You can't always compute exact numbers from a diff alone — flag anything that looks under-tested (a new branch, error path, or edge case with no corresponding test) rather than trying to produce a precise percentage.
- If a new backend feature was added but the root `test:backend:coverage` command wasn't extended to include its packages, flag it — easy to miss, explicitly called out in AGENTS.md.

**Migrations** (if the diff touches `migrations/`)
- New migration file, not an edit to one already merged/applied — check `git log` on the file if unsure whether it's new.
- Has both `-- +goose Up` and `-- +goose Down` sections, and the down actually reverses the up.

**Documentation**
- Compare what the diff changes against `README.md` and `docs/*.md`. If the change would make something a new contributor reads there stale or wrong, the same PR should update it — this is an explicit AGENTS.md rule. If genuinely nothing applies, that's fine, just don't skip the check.
- If the PR body's "Qué se hizo" doesn't match what the diff actually does (scope creep, or a described change that isn't there), call that out specifically — it's the fastest way to catch a PR solving a different problem than intended.

**PR hygiene** (PR mode: check the fetched title/body; local mode: check what's drafted so far if anything)
- Title: Conventional Commits (`<type>: <description>`), English, imperative, lowercase, no trailing period.
- Body: the exact four headings from `AGENTS.md` (Qué se hizo / Cómo probarlo / Cambios en base de datos / Capturas) — don't accept paraphrased headings.
- `Tarea: #N` present if the branch name carries an issue number, absent if it doesn't (don't demand one that shouldn't exist).

**Dependencies**
- Any new package addition is justified by a concrete need (AGENTS.md: "keep dependencies minimal"). A new dependency for something a few lines of existing code could do is worth flagging.

**Size and shape**
- If the diff is large enough that a careful line-by-line review isn't realistic (rough rule of thumb: several hundred+ changed lines spanning unrelated concerns), say so up front and suggest splitting, rather than skimming it and producing a shallow review that looks thorough.
- Dead code, commented-out blocks, obvious duplication of logic that already exists elsewhere in the codebase.

**CI status** (PR mode only, optional)
- If useful, mention whether CI is green using the same approach as the `ci-debug` skill — don't re-run the whole suite yourself as part of this review, that skill already owns that job. A one-line status note is enough context for the review; if checks are failing, say so and point at `ci-debug` for the diagnosis rather than duplicating it here.

## 6. Draft the review — always show it before posting anything

Produce two things:

1. **A summary** (a few sentences to a short paragraph): overall assessment, the most important issues first, and whether documentation needs updating. This is not a PR body — it's your review commentary.
2. **Inline comments**: a list of `{file, line, comment}` for anything specific enough to attach to a line — a missing test for a new branch, a naming mismatch, a layering violation, a stale doc reference. Use the line number from the diff (the new/right-hand side for additions, matching what GitHub's line-comment API expects). Don't force an inline comment for things that are really about the PR as a whole (missing doc update, PR body mismatch, overall size) — those belong in the summary.

Show both to the user in the chat, exactly as they'd be posted, before touching GitHub. Wait for explicit confirmation. This mirrors how `open-pr` never pushes without showing the diff first — a review comment on a real PR is the same kind of one-way door, so don't skip this even if the findings seem minor or obviously correct.

In local-diff mode there's nothing to post — the draft above *is* the deliverable. Stop here.

## 7. Post the review (PR mode only, after confirmation)

Post as a single review with event `COMMENT` — **never** `APPROVE` or `REQUEST_CHANGES`. This skill's job is to surface things a human reviewer should look at, not to substitute for the human approval the repo's branch protection rules require. If everything looks solid, say so in the summary text, but still post as `COMMENT` (or just report "sin observaciones" in chat and skip posting anything if there's truly nothing to say).

```bash
curl -s -X POST -H "Authorization: token $TOKEN" \
  "https://api.github.com/repos/LoteoApp/LoteosAPP/pulls/<pr_number>/reviews" \
  -d '{
    "event": "COMMENT",
    "body": "<summary from step 6>",
    "comments": [
      {"path": "<file>", "line": <line>, "side": "RIGHT", "body": "<comment>"}
    ]
  }'
```

`side: "RIGHT"` refers to the new version of the file (additions/context) — use `"LEFT"` only if commenting on a removed line, which is rare for this kind of review. If a comment's line isn't actually part of the diff hunk (context lines outside what changed), GitHub's API will reject it — double check the line falls within a `patch` hunk from step 2 before including it.

Report the review URL back to the user when done.
