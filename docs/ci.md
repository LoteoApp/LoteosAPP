# Integración continua

## Cuándo corre

El workflow `.github/workflows/ci.yml` corre:

- en cada pull request, sin importar la rama base;
- en cada push directo a `main` o `develop`.

`dependency-review` es la excepción: solo corre en pull requests, porque
compara las dependencias de la rama base contra las del head del PR y en un
push directo no hay una base con la que comparar.

## Jobs

| Job | Qué hace |
| --- | --- |
| `build` | Typecheck, lint y build del frontend; `go vet` y build del backend. |
| `test` | `pnpm test` (suite de frontend y backend, sin cobertura). |
| `coverage` | `pnpm test:coverage`, aplicando los umbrales de cobertura definidos en `AGENTS.md` (`scripts/check-go-coverage.mjs` en el backend). |
| `dependency-audit` | `pnpm audit` (frontend) y `govulncheck` (backend), buscando vulnerabilidades conocidas en todo el árbol de dependencias actual. |
| `dependency-review` | Acción oficial de GitHub que revisa solo las dependencias nuevas o modificadas en el diff del PR. |
| `compose-config` | `docker compose config`: valida sintaxis y variables de `compose.yaml` sin construir ni levantar contenedores. |

`build`, `test`, `coverage` y `dependency-audit` corren en paralelo, cada uno
en su propio runner.

`test` y `coverage` levantan además un servicio `postgis/postgis` y aplican las
migraciones con `go run ./cmd/migrate` antes de correr la suite. Es PostGIS y no
PostgreSQL a secas porque el modelo de entidades guarda geometría: sin la
extensión, los tests de integración del repositorio se saltarían y una consulta
rota pasaría inadvertida.

## Relación entre `dependency-audit` y `dependency-review`

Cubren cosas distintas y no son redundantes:

- `dependency-review` solo mira lo que cambia en el PR puntual. Si una
  dependencia nueva o actualizada tiene una vulnerabilidad conocida, frena
  ahí.
- `dependency-audit` escanea todo el árbol de dependencias actual, haya
  cambiado algo en el PR o no. Detecta vulnerabilidades publicadas después de
  que una dependencia ya existente fue agregada.
- `govulncheck`, dentro de `dependency-audit`, además hace análisis de
  alcance de código: solo reporta vulnerabilidades de funciones que el código
  del proyecto realmente llama, incluyendo la librería estándar de Go (no
  solo módulos de terceros).

## Versión de Go en CI

`build`, `test`, `coverage` y `dependency-audit` usan la versión de Go
declarada en la directiva `go` de `apps/backend/go.mod` (vía
`actions/setup-go` con `go-version-file`). Mantener esa directiva en la
última versión de parche disponible evita que `govulncheck` reporte
vulnerabilidades ya corregidas en la librería estándar.

## Secretos y credenciales

El repo tiene activado el secret scanning nativo de GitHub con push
protection (**Settings > Code security**). Push protection corre en el
servidor de GitHub y rechaza el `push` antes de que un secreto conocido
(tokens de AWS, Stripe, npm, GitHub, etc.) llegue al historial; secret
scanning además revisa el repo de forma continua y avisa por alerta si algo
se filtró igual. No es un job de este workflow ni vive en ningún archivo del
repositorio — se administra desde GitHub, y aplica antes de que el código
llegue a un PR, no como parte de este pipeline.

## Protección de ramas

`develop` y `main` requieren, para poder mergear un PR:

- al menos 1 aprobación de un reviewer;
- que los 6 checks de este workflow estén en verde;
- que la rama del PR esté actualizada contra la base (`strict: true`).

Esta regla no vive en ningún archivo del repositorio: se administra desde
GitHub, en **Settings > Branches > Branch protection rules**. Cualquier
cambio (agregar un check nuevo, sumar otro colaborador, exigirlo también a
admins) se hace ahí, no con un commit.

Los tres colaboradores del repo son admins, y la regla está configurada sin
`enforce_admins`. Esto significa que, aunque la regla se muestra y aplica
igual para todos, cualquier admin puede optar por "Merge without waiting for
requirements" y saltear la aprobación o el CI si lo necesita — queda
registrado en el historial del PR, pero no es un bloqueo absoluto.

## Origen permitido de los PRs hacia main

GitHub no tiene una regla nativa de protección de rama para restringir desde
qué rama puede venir un PR (ni en branch protection clásica ni en Rulesets).
El job `main-source-branch` cubre ese hueco: corre solo en PRs contra `main`
y falla si la rama de origen no es `develop` ni empieza con `hotfix/`.

`hotfix/*` es la vía de escape para arreglos urgentes que no pueden esperar a
pasar por `develop` primero. Si se mergea un hotfix a `main`, hay que
mergearlo (o cherry-pickearlo) también a `develop` en el mismo momento, para
que no se pierda en el próximo release normal. Esa sincronización es manual,
el CI no la fuerza.
