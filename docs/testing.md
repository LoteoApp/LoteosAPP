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
| `postgres.TestLoteoRepository` | `DATABASE_URL` | Alta de loteo con plano, actualización de lote/manzana/calle, consulta de asignación y registro concurrente del DXF. Verifica la geometría PostGIS, que exista un solo archivo DXF activo, el alta/listado/descarga/baja de fotos y planos a nivel loteo y lote, que el DXF y un documento_legal sean inalcanzables por ese flujo, y que el cupo de 20 por entidad se respete bajo carga concurrente. |
| `postgres.TestReservationRepository` | `DATABASE_URL` | Alta, alcance, idempotencia, cancelación y vencimiento de reservas contra PostgreSQL real, incluyendo la consistencia con el estado e historial del lote. |
| `postgres.TestSaleRepository*` | `DATABASE_URL` | Alta de ventas al contado y financiadas, plan de pago y cuotas persistidas, idempotencia y alcance por agencia. |
| `postgres.TestConvertReservation*`, `postgres.TestSaleReservaLinkConstraints` | `DATABASE_URL` | Conversión de reserva en venta en las tres modalidades, vínculo en ambos sentidos, vendedor y alcance (administrador ajeno, colega de agencia, otra agencia, agencia desasignada, actor que cambia de rol antes de que la conversión lo bloquee), vencimiento exacto y reloj leído después de esperar el lock del lote, reserva cancelada/vencida/convertida, reserva que no es la que tiene reservado el lote, cliente o lote incompletos, idempotencia (replay con otro precio, otras condiciones, clave de una venta ordinaria), rollback completo con un fallo inyectado después de escribir, reconciliación de un `COMMIT` ambiguo, carreras (dos conversiones, misma clave, contra cancelación y contra el worker) y las restricciones del vínculo. |
| `migrate.TestSaleReservationLinkMigration` | `DATABASE_URL` | Aplica `00015` en un schema descartable sobre datos previos, verifica `NULL` en las ventas existentes, la FK compuesta, la unicidad por reserva aunque la venta se cancele, la inmutabilidad del vínculo y el `Down`. |
| `postgres.TestCollectionRepository*` | `DATABASE_URL` | Estado de deuda, cobro de entrega y cuotas en orden, rechazo de cobros repetidos o salteados, cancelación total con saldo desactualizado, cierre de la venta y finalización del lote con origen `cobranza`, listado de vencimientos con filtros, alcance y paginación por ids, una cuota que vence hoy sigue pendiente hasta el día siguiente en hora de Argentina, el estado de deuda leído en una sola instantánea aunque un cobro confirme a mitad de la lectura, y cargos adicionales en otra moneda (totales por moneda y rollback del cobro si un cargo es inválido). |
| `r2.TestClientIntegration` | `CLOUDFLARE_R2_*` | Sube, lee y borra un objeto en el bucket, bajo el prefijo `integration-test/`. |

Los dos tests de PostgreSQL borran lo que crearon. El test de R2 escribe en el
bucket del entorno y también limpia antes de terminar; no correrlo apuntando a
un bucket de producción.

La suite `postgres.TestLoteoRepository` también cubre el estado inicial, las
transiciones atómicas, los conflictos por estado esperado obsoleto y dos
compare-and-set concurrentes sobre el mismo lote. La suite
`migrate.TestEntityModelStateHistory` aplica las migraciones en un schema
descartable y verifica el backfill, los triggers, la justificación obligatoria
y que el historial del lote sea append-only.

La suite del worker usa un proceso falso y contextos con deadline para verificar
la ejecución inmediata, el lote configurado, los fallos y el apagado sin sleeps
de duración comercial. Las pruebas de reservas de PostgreSQL deben ejecutarse
con las migraciones aplicadas y no se consideran realizadas cuando falta
`DATABASE_URL`.

Las pruebas de conversión crean sus propios loteos, usuarios y reservas y los
borran al terminar; el cleanup (`deleteLoteo`) borra cobros, cuotas, planes y
ventas antes que las reservas por la FK `ventas.reserva_id`. No modifican el
esquema ni deshabilitan triggers fuera de ese cleanup: el fallo a mitad de la
transacción se inyecta con el hook `afterConversionWrite` del repositorio
(expuesto solo a los tests por `export_test.go`). Las carreras arrancan juntas
detrás de un canal y aceptan cualquiera de los dos órdenes; además, el mismo hook
pausa una conversión con todos sus locks tomados para forzar el orden en que la
cancelación o el worker (que ya listó la reserva como candidata) esperan y
después ven la reserva `convertida`. La prueba del reloj y las de orden forzado
esperan a ver la otra sesión bloqueada con `pg_blocking_pids` en lugar de dormir.
Como `ExpireDue` procesa todas las reservas vencidas de la base, las pruebas
que lo llaman usan instantes ya pasados, para no vencer reservas vigentes de
otros usuarios de la base compartida. Las demás crean reservas con fechas
futuras, así el worker real del entorno no las vence en medio de la prueba.

## Prueba manual del alta de loteo

El alta (`POST /api/v1/loteos`), el listado y el detalle ya tienen pantalla; el
detalle permite editar los datos manuales de lotes, manzanas y calles según el
rol del usuario. `scripts/smoke-loteos.sh`
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
