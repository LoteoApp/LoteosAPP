---
name: create-tasks
description: >
  Turns one or more feature descriptions into GitHub issues in the
  LoteoApp/LoteosAPP repo, following the project's epic/task hierarchy:
  native GitHub sub-issues plus custom `epic`/`task` labels, added to the
  org's Project board. Use this whenever the user describes new work to plan
  (a feature, a set of related changes, "necesitamos hacer X e Y") and wants
  it tracked as issues — not just when they say "épica" or "tarea" literally.
  Also use it when the user wants a new task added under work that's already
  in flight. Always check existing epics first and either fit the new work
  into one or propose a new epic — don't create either blindly.
compatibility: Requires a GitHub personal access token with repo and project
  scope (ask the user if one hasn't been shared in the session).
---

# LoteosAPP: create epics and tasks from a feature description

This repo tracks work as GitHub issues in two tiers: an "Épica" issue (label
`epic`) groups related work, and "task" issues (label `task`) are its native
GitHub sub-issues. Both get added to the org Project board
(`https://github.com/orgs/LoteoApp/projects/2`). See
`references/github-ids.md` for the cached node IDs the API calls below need.

## 1. Read the existing epics first

```bash
curl -s -H "Authorization: token $TOKEN" \
  "https://api.github.com/repos/LoteoApp/LoteosAPP/issues?labels=epic&state=all&per_page=50"
```

For each open epic that looks related, pull its sub-issues too
(`GET /repos/LoteoApp/LoteosAPP/issues/{number}/sub_issues`) to see what's
already planned under it — the new task might already exist, or might belong
there.

## 2. Decide: existing epic or new one — and say so before creating anything

This is a judgment call, not a mechanical one. An epic should be a coherent
unit of scope, not just "whatever the user mentioned in one sentence" — e.g.
router setup and the base layout were split into two *tasks* under one epic
("Layout del frontend") rather than two separate epics, because they serve
the same end (real navigation), not because router and layout happen to be
different files.

Before creating or reusing an epic, tell the user your plan in one or two
lines ("Esto encaja en la épica #78 Layout del frontend" / "Esto no encaja
en ninguna épica actual, propongo una nueva: Épica: X") and let them correct
it. Users tend to have opinions about epic boundaries — don't skip this
check even if the fit seems obvious.

## 3. Create the epic (only if needed)

```bash
curl -s -X POST https://api.github.com/repos/LoteoApp/LoteosAPP/issues \
  -H "Authorization: token $TOKEN" -H "Accept: application/vnd.github+json" \
  -d '{"title": "Épica: <nombre>", "body": "<qué abarca, 1-2 frases>", "labels": ["epic"]}'
```

Title is always `Épica: <nombre>`, in Spanish, matching the existing three
(`Configuración inicial`, `CI/CD`, `Layout del frontend`).

## 4. Create each task as a sub-issue

Create the issue first:

```bash
curl -s -X POST https://api.github.com/repos/LoteoApp/LoteosAPP/issues \
  -H "Authorization: token $TOKEN" -H "Accept: application/vnd.github+json" \
  -d '{"title": "<tarea, en español, verbo en infinitivo>", "body": "Fuente: épica #<epic_number>. <qué hay que hacer>.", "labels": ["task"], "assignees": ["<github-username>"]}'
```

Note the body convention: starts with `Fuente: épica #<n>.` so anyone
reading the task alone knows which epic it belongs to, even outside the
sub-issue UI.

Then link it as a sub-issue of the epic. This needs the task's numeric
database **`id`** field from the create response (not its `number`):

```bash
curl -s -X POST https://api.github.com/repos/LoteoApp/LoteosAPP/issues/<epic_number>/sub_issues \
  -H "Authorization: token $TOKEN" -H "Accept: application/vnd.github+json" \
  -d '{"sub_issue_id": <task_id>}'
```

## 5. Add both the epic (if new) and each new task to the Project board

GraphQL, one call per issue, using the issue's `node_id` (from the same
create response) and the cached project ID:

```bash
curl -s -X POST https://api.github.com/graphql \
  -H "Authorization: bearer $TOKEN" \
  -d '{"query":"mutation { addProjectV2ItemById(input: {projectId: \"PVT_kwDOEoypDc4BfWHQ\", contentId: \"<issue_node_id>\"}) { item { id } } }"}'
```

The Project board card is just a view onto the linked issue — it has no
separate title/description/labels of its own. Whatever `body` and `labels`
you set when creating the issue in steps 3-4 is exactly what shows up on the
card, so never create an issue with an empty body or a missing `epic`/`task`
label "to fill in later": there is no later, the card only ever reflects
what the issue already has. Double-check both fields are set before moving
on to this step.

## 6. Assign, and offer to branch

If the user asked to be assigned (or it's implied — "que te la asigne"), set
`assignees` on creation as shown above, or `PATCH` an existing issue's
`assignees` field.

If the user wants to start work right away on one of the tasks, hand off to
the `open-pr` skill's branch step (step 1: branch from a freshly pulled
`develop`, named `<type>/<task_number>-<slug>`) rather than duplicating that
logic here.

## 7. Report back

Summarize what was created: epic (new or existing, with number), each task
with its number and URL, confirmation they're on the Project board. Don't
just say "listo" — the user has repeatedly cared about being able to see
this structure reflected correctly on the board.
