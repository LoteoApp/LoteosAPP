# Pruebas y cobertura

## Herramientas elegidas

### Backend Go

Se usa la librería estándar de Go:

- `testing` para pruebas unitarias y de integración;
- `net/http/httptest` para probar handlers mediante requests y responses reales;
- `go test -coverprofile` y `go tool cover` para medir cobertura.

Es la opción idiomática para Go, no agrega dependencias y permite probar el
contrato observable del código. Los servicios se prueban con fakes pequeños de
las interfaces que consumen. No se deben simular los detalles internos de
`pgx`; los repositorios SQL se prueban como integración contra PostgreSQL real.

Cuando las primeras funcionalidades de negocio incorporen repositorios SQL, se
evaluará `testcontainers-go` para levantar una instancia aislada de PostgreSQL
por suite. No se incorpora todavía porque el único repositorio actual es el
diagnóstico del entorno y Compose ya permite su verificación integrada.

### Frontend React

Se usan:

- Vitest como runner integrado con Vite y TypeScript;
- React Testing Library para renderizar y consultar la UI;
- `@testing-library/jest-dom` para aserciones del DOM;
- `@testing-library/user-event` para interacciones de usuario;
- jsdom como entorno de navegador para pruebas unitarias y de componentes;
- cobertura V8 mediante `@vitest/coverage-v8`.

Las pruebas deben observar lo mismo que una persona: textos, roles, labels,
estados de carga, errores y resultados. No deben depender del estado interno de
React, nombres de funciones privadas ni clases de Tailwind.

## Umbrales

Para código nuevo o modificado dentro de una funcionalidad:

| Métrica | Mínimo |
| --- | ---: |
| Líneas | 80% |
| Sentencias | 80% |
| Funciones | 80% |
| Ramas | 75% |
| Reglas críticas del negocio | 90% recomendado |

Vitest aplica los umbrales por archivo sobre `src/features`. En Go, `go tool
cover` solo imprime porcentajes, así que quien aplica el umbral es
`scripts/check-go-coverage.mjs`: lee `apps/backend/coverage.out`, agrega las
sentencias por paquete y falla si un paquete —o el total— queda por debajo del
80%. Go no reporta ramas; la cobertura de sentencias hace las veces de líneas y
sentencias. Al agregar una feature backend, se debe incorporar su paquete al
comando raíz `test:backend:coverage:profile`.

Los paquetes cuyos tests necesitan un servicio externo se miden solo cuando ese
servicio está configurado (`DATABASE_URL` para `repository/postgres`,
`CLOUDFLARE_R2_ENDPOINT` para `storage/r2`): sin él sus tests de integración se
saltan y el número hablaría del entorno, no del código. El job de cobertura de
CI levanta PostGIS y aplica las migraciones, así que ahí el umbral sí corre
sobre el repositorio.

La cobertura no reemplaza la calidad de los casos. Cada funcionalidad debe
probar, cuando corresponda:

- comportamiento exitoso;
- validaciones y límites;
- errores de dependencias;
- permisos y autenticación;
- estados de carga, vacío y error en la UI;
- regresiones de bugs corregidos.

## Comandos

Desde la raíz:

```powershell
# Backend y frontend
pnpm test

# Reportes de cobertura de ambos proyectos
pnpm test:coverage

# Suites individuales
pnpm test:backend
pnpm test:frontend

# Cobertura individual
pnpm test:backend:coverage
pnpm test:frontend:coverage
```

Modo watch del frontend:

```powershell
pnpm --filter @loteos/frontend test:watch
```

## Tests de integración

Los tests que necesitan un servicio real se saltan solos cuando faltan sus
variables de entorno, así que `pnpm test` pasa sin ellos en local. En CI los de
PostgreSQL sí corren: los jobs de test y cobertura levantan un servicio
`postgis/postgis`, aplican las migraciones con `go run ./cmd/migrate` y exportan
`DATABASE_URL`. Para ejecutarlos localmente, correr la suite con los secrets
inyectados:

```powershell
doppler run -- pnpm test:backend
```

| Test | Necesita | Qué hace |
| --- | --- | --- |
| `postgres.TestUserRepository` | `DATABASE_URL` | SQL real contra la base de Supabase con las migraciones aplicadas. |
| `postgres.TestLoteoRepository` | `DATABASE_URL` | Alta de loteo con plano, actualización de lote, consulta de asignación y registro concurrente del DXF. Verifica la geometría PostGIS y que exista un solo archivo DXF activo. |
| `r2.TestClientIntegration` | `CLOUDFLARE_R2_*` | Sube, lee y borra un objeto en el bucket, bajo el prefijo `integration-test/`. |

Los dos tests de PostgreSQL borran lo que crearon. El test de R2 escribe en el
bucket del entorno y también limpia antes de terminar; no correrlo apuntando a
un bucket de producción.

## Prueba manual del alta de loteo

El alta (`POST /api/v1/loteos`), el listado y el detalle ya tienen pantalla; la
carga de datos de un lote (`PATCH .../lotes/{loteId}`) todavía no. `scripts/smoke-loteos.sh`
prueba el recorrido completo por HTTP como verificación end-to-end
independiente del frontend: login contra Supabase Auth, alta de un loteo con
plano, carga de datos de un lote y los caminos de error (número repetido, lote
de otro loteo, referencia a una manzana que no está en el plano, anillo
abierto, request sin token).

Levantar el backend en una terminal:

```powershell
cd apps/backend
doppler run -- go run ./cmd/server
```

Y en otra, desde la raíz del repositorio:

```powershell
$env:ADMIN_EMAIL="<cuenta administrador>"
$env:ADMIN_PASSWORD="<contraseña>"
doppler run -- bash scripts/smoke-loteos.sh
```

Necesita `jq` y `curl`. La contraseña se pasa por variable de entorno y no se
escribe a ningún archivo.

El script **deja creado** el loteo para poder inspeccionarlo e imprime su id al
terminar. Para borrarlo, con el id que imprimió:

```powershell
doppler run -- psql $env:DATABASE_URL -f scripts/smoke-loteos-cleanup.sql -v loteo_id="'<uuid>'"
```

El reporte HTML del frontend se genera en
`apps/frontend/coverage/index.html`. El perfil de Go se genera en
`apps/backend/coverage.out`; ambos están ignorados por Git.

## Ubicación y nombres

- Go: archivo `*_test.go` junto al paquete probado.
- React y TypeScript: archivo `*.test.ts` o `*.test.tsx` junto al módulo o
  componente probado.
- Utilidades globales del entorno frontend: `apps/frontend/src/test`.

No se crean directorios globales con tests desconectados de la funcionalidad.
