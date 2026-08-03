# LoteosAPP

## Package manager

- Never use `npm`.
- Always use `pnpm` for JavaScript and frontend package-management commands.
- Use `pnpm install`, `pnpm add`, `pnpm remove`, `pnpm run`, and `pnpm dlx` as applicable.
- Do not create or commit `package-lock.json` or `npm-shrinkwrap.json`; the workspace lockfile is `pnpm-lock.yaml`.
- If documentation shows an `npm` command, translate it to the equivalent `pnpm` command before running it.

## Repository layout

- `apps/frontend`: React + TypeScript + Vite + Tailwind CSS.
- `apps/backend`: Go HTTP API.
- `migrations`: Versioned PostgreSQL SQL migrations managed by Goose.
- Keep frontend and backend independently buildable.

## Architecture

- Organize backend and frontend code by feature, not by global technical layers.
- Follow the dependency rules documented in `docs/architecture.md`.
- Backend feature cores must not depend on HTTP, PostgreSQL, or concrete `pgx` types.
- Keep backend infrastructure under `internal/platform` and dependency wiring under `internal/app`.
- Keep frontend composition under `src/app`, feature code under `src/features`, and genuinely reusable code under `src/shared`.
- Frontend features may depend on `shared`; they must not import another feature's internal files.
- Prefer direct imports over broad barrel files.
- Do not create generic `utils`, `helpers`, `common`, `controllers`, `services`, `repositories`, or `models` directories.
- Add abstractions and dependencies only when a concrete use case requires them; do not create empty placeholder directories.

## Backend and database

- Use `github.com/jackc/pgx/v5/pgxpool` for concurrent PostgreSQL access from the API.
- Keep migrations as SQL files under `migrations/` with Goose `-- +goose Up` and `-- +goose Down` sections.
- Do not edit a migration after it has been applied; add a new migration instead.
- Use `docker compose up --build` to start PostgreSQL, migrations, backend, and frontend together.

## Testing and coverage

- Every new behavior and every bug fix must include tests in the same change.
- Backend tests use Go's standard `testing` package and `net/http/httptest`; do not add assertion libraries without a concrete need.
- Test backend business services with small fakes of their required interfaces, HTTP handlers through their observable request/response contract, and PostgreSQL repositories with integration tests against a real PostgreSQL instance.
- Frontend tests use Vitest, React Testing Library, `@testing-library/jest-dom`, and `@testing-library/user-event` for user interactions.
- Frontend tests must query and assert the UI as a user would; avoid testing internal state, implementation details, Tailwind classes, or private functions.
- Keep tests next to the code under test using `*_test.go`, `*.test.ts`, or `*.test.tsx`.
- Changed feature code must maintain at least 80% lines, statements, and functions, plus at least 75% branches. Critical business rules should maintain at least 90% coverage.
- Coverage is a safety signal, not the objective: include successful behavior, validation, error paths, permissions, and boundary cases relevant to the change.
- Do not lower coverage thresholds, remove assertions, add broad exclusions, or skip tests merely to make a suite pass.
- When adding a backend feature, extend the root `test:backend:coverage` command so its core and HTTP packages are included in the coverage report.
- Run `pnpm test` for all suites and `pnpm test:coverage` for coverage before finishing a functional change.

## Verification

- Frontend: `pnpm --filter @loteos/frontend typecheck`, `pnpm --filter @loteos/frontend test`, and `pnpm --filter @loteos/frontend build`.
- Backend: `go test ./...` and `go vet ./...` from `apps/backend`.
- Full test suite and coverage: `pnpm test` and `pnpm test:coverage` from the repository root.
- Database environment: `docker compose config` and `docker compose up --build`.
- Keep dependencies and configuration minimal; do not add libraries without a concrete need.
