# LoteosAPP

## Language

- Respond to the user in Spanish.
- Code, identifiers, comments, and commit messages stay in English.

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

- Organize frontend code by feature. Organize backend code by technical layer
  (`internal/business`, `internal/infrastructure`), not by feature — except
  `internal/business/usecase` and `internal/infrastructure/delivery/webapp/dto`,
  which group use cases and HTTP DTOs into per-feature subpackages (e.g.
  `usecase/users`, `usecase/system`, `dto/users`, `dto/system`).
- Follow the dependency rules documented in `docs/architecture.md`.
- Backend domain and use cases (`internal/business`) must not depend on HTTP,
  PostgreSQL, or concrete `pgx` types.
- Business errors a use case returns to a caller are `*domain.Error` (`Kind`,
  `Code`, `Message`, optional `Cause`), not `errors.New(...)`. `Kind` is a
  business classification (`KindInvalid`, `KindForbidden`, `KindConflict`,
  `KindNotFound`, `KindUnavailable`), never an HTTP status. Set `Cause` to
  the underlying error (e.g. a PostgreSQL failure) when one exists, so it can
  be logged without exposing it in `Message`. `response.WriteError` is the
  single place that maps `Kind` to an HTTP status, writes `Code`/`Message`,
  and logs `Cause`; handlers never write their own error-mapping `switch`.
  Errors that aren't `*domain.Error` (unexpected failures) are logged and
  hidden behind a generic 500.
- Keep domain entities under `internal/business/domain`, contracts the
  business needs from its adapters (e.g. `Repository`) under
  `internal/business/gateway`, and use cases under `internal/business/usecase`.
  Use cases depend on `gateway` contracts, never on a concrete adapter. Each
  use case is a single-method `Execute` interface plus its implementation,
  defined together in one file under its feature subpackage (e.g.
  `usecase/users/create_user.go`).
- Keep every adapter (persistence, environment configuration, HTTP delivery,
  HTTP server bootstrap) under `internal/infrastructure`.
- Each HTTP route gets its own handler with a single use case as its only
  dependency (e.g. `CreateUserHandler` depends only on `users.CreateUser`);
  do not group multiple routes into one handler struct. A handler implements
  `handler.HTTPHandler` (`Handle(w, r) error`) instead of writing its own
  error response: it returns the use case's error (or its own `decodeJSON`
  error) and lets `handler.Adapt` translate it via `response.WriteError`.
  `route.go` registers `handler.Adapt(h)`, not the handler's method directly.
  A handler with no failure path (e.g. `Live`) stays a plain
  `func(w, r)` — don't wrap it in a struct just to satisfy the interface.
- Keep HTTP request/response structs out of `handler`; define them under
  `internal/infrastructure/delivery/webapp/dto/<feature>` instead. Each `dto`
  subpackage declares `package dto` regardless of its directory name, so it
  never collides with the matching `usecase` subpackage when both are
  imported in the same handler file.
- Decode request bodies with the shared generic `decodeJSON[T]` helper in
  `internal/infrastructure/delivery/webapp/handler` instead of repeating
  `json.NewDecoder(...).Decode(...)` per handler; it returns a `*domain.Error`
  on failure instead of writing the response itself.
- Keep the dependency injection container (IoC wiring: pool, repositories, use
  cases, handlers) under `internal/infrastructure/delivery/webapp/dependencies`.
  It only builds objects; it does not read configuration, register routes, or
  build the HTTP server.
- `internal/app` is the composition root and owns process lifecycle: it reads
  configuration, asks `dependencies` for the object graph, registers routes,
  builds the HTTP server, then runs and shuts it down.
- Keep frontend composition under `src/app`, feature code under `src/features`, and genuinely reusable code under `src/shared`.
- Frontend features may depend on `shared`; they must not import another feature's internal files.
- The app must work on phones: build new frontend UI mobile-first. Write unprefixed Tailwind classes for the smallest viewport first, then layer `sm:`/`md:`/`lg:` overrides for larger screens, not the other way around. See `docs/architecture.md` for more detail.
- Prefer direct imports over broad barrel files.
- Do not create generic `utils`, `helpers`, `common`, `controllers`, `services`, `repositories`, or `models` directories.
- Add abstractions and dependencies only when a concrete use case requires them; do not create empty placeholder directories.

## Code comments

- Don't add comments that just restate what the code already does, or that narrate the design decision behind it (why this style, what alternatives were considered, what it's meant to achieve). That kind of context belongs in `docs/` or the PR description, not inline in the source.
- Reserve comments for the rare case where behavior genuinely isn't obvious from reading the code: a non-obvious workaround, a business or regulatory rule that isn't visible in the code itself, a warning about a footgun.
- When tempted to explain what a block of code does, prefer a clearer name or a smaller function first; reach for a comment only if that isn't enough.

## Backend and database

- Use `github.com/jackc/pgx/v5/pgxpool` for concurrent PostgreSQL access from the API.
- Keep migrations as SQL files under `migrations/` with Goose `-- +goose Up` and `-- +goose Down` sections.
- Do not edit a migration after it has been applied; add a new migration instead.
- Use `docker compose up --build` to start PostgreSQL, migrations, backend, and frontend together.

## Testing and coverage

- Every new behavior and every bug fix must include tests in the same change.
- Backend tests use Go's standard `testing` package and `net/http/httptest`; do not add assertion libraries without a concrete need.
- Test backend use cases with small fakes of their required `gateway` interfaces, HTTP handlers through their observable request/response contract, and PostgreSQL repositories with integration tests against a real PostgreSQL instance.
- Keep fakes for `gateway` contracts in `internal/business/gateway/gatewayfake`, next to the interfaces they implement, so every `usecase` subpackage reuses them instead of redefining fakes per feature.
- Frontend tests use Vitest, React Testing Library, `@testing-library/jest-dom`, and `@testing-library/user-event` for user interactions.
- Frontend tests must query and assert the UI as a user would; avoid testing internal state, implementation details, Tailwind classes, or private functions.
- Keep tests next to the code under test using `*_test.go`, `*.test.ts`, or `*.test.tsx`.
- Changed feature code must maintain at least 80% lines, statements, and functions, plus at least 75% branches. Critical business rules should maintain at least 90% coverage.
- Coverage is a safety signal, not the objective: include successful behavior, validation, error paths, permissions, and boundary cases relevant to the change.
- Do not lower coverage thresholds, remove assertions, add broad exclusions, or skip tests merely to make a suite pass.
- When adding a backend feature, extend the root `test:backend:coverage` command so its core and HTTP packages are included in the coverage report.
- Run `pnpm test` for all suites and `pnpm test:coverage` for coverage before finishing a functional change.

## Commits

- Use simplified Conventional Commits: `<type>: <description>`.
- Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`.
- Description in English, imperative, lowercase, no trailing period.
- Examples:
  - `feat: add authentication`
  - `fix: correct installment calculation`
  - `refactor: reorganize user service`
  - `test: add login tests`
  - `docs: document customer endpoint`
  - `chore: update dependencies`

## Pull Requests

GitHub pre-carga `.github/pull_request_template.md` al abrir un PR desde la web.
Desde la CLI se pasa explícitamente:

```powershell
gh pr create --template .github/pull_request_template.md
```

Every PR description must include:

```
## Qué se hizo

Descripción breve del cambio.

## Cómo probarlo

Pasos para levantar el entorno y probar el cambio.

## Cambios en base de datos

Migraciones agregadas o modificadas, si aplica.

## Capturas

Agregar capturas si hay cambios visuales.

## Issue

Closes #NNN
```

La línea `Closes #NNN` es obligatoria: sin ella el issue queda huérfano del PR y
hay que actualizar el board a mano. Si el PR avanza un issue pero no lo termina,
usar `Refs #NNN`.

Como los PR van contra `develop`, GitHub no cierra el issue al mergear; lo cierra
recién cuando `develop` llega a `main`. La línea sirve igual, porque deja el PR
enlazado en el issue desde el momento en que se abre.

Antes de pushear, revisar si el cambio deja desactualizada la documentación existente
(`README.md`, `docs/`) y actualizarla en el mismo PR.

## Verification

- Frontend: `pnpm --filter @loteos/frontend typecheck`, `pnpm --filter @loteos/frontend lint`, `pnpm --filter @loteos/frontend test`, and `pnpm --filter @loteos/frontend build`.
- Backend: `go test ./...` and `go vet ./...` from `apps/backend`.
- Full test suite and coverage: `pnpm test` and `pnpm test:coverage` from the repository root.
- Database environment: `docker compose config` and `docker compose up --build`.
- Keep dependencies and configuration minimal; do not add libraries without a concrete need.
- CI runs build, test, coverage, and dependency-audit checks automatically on every pull request; see `docs/ci.md`. Running these commands locally before pushing still saves round trips.
