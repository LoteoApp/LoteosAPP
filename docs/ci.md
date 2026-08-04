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
| `build` | Typecheck y build del frontend; `go vet` y build del backend. |
| `test` | `pnpm test` (suite de frontend y backend, sin cobertura). |
| `coverage` | `pnpm test:coverage`, aplicando los umbrales de cobertura definidos en `AGENTS.md`. |
| `dependency-audit` | `pnpm audit` (frontend) y `govulncheck` (backend), buscando vulnerabilidades conocidas en todo el árbol de dependencias actual. |
| `dependency-review` | Acción oficial de GitHub que revisa solo las dependencias nuevas o modificadas en el diff del PR. |
| `compose-config` | `docker compose config`: valida sintaxis y variables de `compose.yaml` sin construir ni levantar contenedores. |

`build`, `test`, `coverage` y `dependency-audit` corren en paralelo, cada uno
en su propio runner.

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
