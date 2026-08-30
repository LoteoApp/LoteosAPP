# PostgreSQL y migraciones

## Conexión recomendada desde Go

Para la API se usa `github.com/jackc/pgx/v5/pgxpool`.

Es la opción recomendada porque:

- Está diseñada para PostgreSQL y sus características específicas.
- Administra un pool de conexiones seguro para múltiples requests concurrentes.
- Permite configurar límites, tiempos de vida, conexiones mínimas y health checks.
- Mantiene las queries parametrizadas con placeholders `$1`, `$2`, etc.

La conexión del backend se configura en
`apps/backend/internal/infrastructure/repository/postgres/pool.go` y se
construye desde `apps/backend/internal/app/app.go`. Al iniciar:

1. Lee `DATABASE_URL`.
2. Construye un `pgxpool.Pool`.
3. Configura límites y timeouts.
4. Ejecuta `Ping` antes de aceptar requests.
5. Cierra el pool al terminar el proceso.

Ejemplo de query parametrizada:

```go
row := pool.QueryRow(ctx,
    `SELECT id, name FROM lots WHERE id = $1`,
    lotID,
)
```

No se deben concatenar valores del usuario dentro del SQL.

## Conexión a Supabase

La base de la aplicación es el Postgres administrado del proyecto de
Supabase ([#127](https://github.com/LoteoApp/LoteosAPP/issues/127)), no un
contenedor local. `DATABASE_URL` se lee desde Doppler (ver
[secrets.md](secrets.md)) y apunta al **pooler en modo session**:

```text
postgres://postgres.<project-ref>:<password>@aws-<region>.pooler.supabase.com:5432/postgres?sslmode=require
```

- **Modo session, no transaction**: el modo transaction (puerto `6543`) no
  soporta prepared statements, que `pgx` usa por default; el modo session
  (puerto `5432`) sí.
- **No la conexión directa**: es IPv6-only en el plan free del proyecto.
- **`sslmode=require`, no `verify-full`**: cifra la conexión pero no valida
  la CA del certificado del pooler, que no es una CA pública. `verify-full`
  necesitaría distribuir el root cert de Supabase como un secreto más; para
  el tamaño de este proyecto no vale la complejidad operativa extra. Nunca
  se omite `sslmode` en la URL: el default de `pgx` (`prefer`) baja a texto
  plano en silencio si la negociación TLS falla.

  **Riesgo aceptado**: sin verificación de CA, un atacante en posición de
  red puede interponerse (MITM) y ver o modificar el tráfico, incluida la
  contraseña de la base. Aceptable mientras el proyecto no maneje datos
  reales de clientes; revisar el paso a `verify-full` con el CA bundle de
  Supabase antes de eso. El riesgo es mayor mientras la conexión use el rol
  administrativo (ver más abajo), así que ambas decisiones se revisan juntas.
- **Rol administrativo, pendiente de separar**: `DATABASE_URL` usa el rol
  `postgres.<project-ref>`, dueño del esquema `public`. El proceso HTTP no
  necesita DDL: corresponde separar un rol de aplicación con permisos
  mínimos del rol que corre las migraciones. Riesgo aceptado por ahora,
  pendiente de resolver antes de manejar datos reales.
- **Pool chico** (`pool.go`): el pooler de Supavisor es compartido entre
  todos los devs en un único proyecto free-tier, a diferencia de un
  contenedor local sin ese límite. `pgxpool` se configura con pocas
  conexiones máximas y libera las inactivas rápido en vez de retenerlas. CI
  no se conecta a esta base: el test de integración en
  `usuario_test.go` se salta si `DATABASE_URL` no está seteada, que es el
  caso hoy en CI.

## Exposición por la Data API y RLS

Supabase publica el esquema `public` por su Data API (PostgREST) con la
`anon` key, que viaja en el bundle del frontend y es pública por diseño. Una
tabla en `public` sin Row Level Security queda legible y escribible por
cualquiera que tenga esa clave, salteando el backend por completo.

`00004_enable_rls_on_public_tables.sql` habilita RLS en `usuarios` y
`goose_db_version` y les revoca los permisos de `anon` y `authenticated`. No
se definen políticas a propósito: toda la lectura y escritura pasa por el
backend, que se conecta como dueño de las tablas y por lo tanto no está
sujeto a RLS. Por la misma razón no debe usarse `FORCE ROW LEVEL SECURITY`,
que también alcanzaría al dueño y dejaría al backend sin acceso.

Regla para tablas nuevas: toda tabla en `public` nace con RLS habilitada en
la misma migración que la crea. Si alguna vez el frontend consulta una tabla
directamente con `supabase-js`, esa tabla necesita además políticas
explícitas; hoy el cliente de Supabase solo se usa para Auth.

Después de agregar tablas conviene revisar los avisos del proyecto
(**Advisors > Security** en el dashboard de Supabase), que marcan
exactamente este problema.

## Validar el entorno

Para confirmar que el backend pudo conectarse a la base durante el
desarrollo, revisar sus logs:

```powershell
docker compose logs backend
```

## Herramienta de migraciones

Se usa [Goose](https://github.com/pressly/goose) con archivos SQL versionados.
El backend usa `pgxpool`; el comando de migraciones usa `database/sql` con el
driver `github.com/jackc/pgx/v5/stdlib`, porque Goose trabaja con `*sql.DB`.

`cmd/migrate` toma un advisory lock de sesión de PostgreSQL
(`goose.WithSessionLocker`) antes de migrar: como `DATABASE_URL` apunta a la
base compartida de Supabase y no a un contenedor descartable, dos `docker
compose up` corriendo a la vez (o un dev compitiendo con CI) deben serializar
en vez de correr `goose_db_version` en paralelo.

Las migraciones están en:

```text
migrations/
├── 00001_init.sql
├── 00002_create_usuarios.sql
├── 00003_rename_keycloak_id_to_auth_provider_id.sql
├── 00004_enable_rls_on_public_tables.sql
└── 00005_create_entity_model.sql
```

`00005` crea el esquema del diagrama v3 (territorio, DXF/PostGIS, comercial
y estados) y habilita RLS en cada tabla nueva, igual que `00004`. No edita
migraciones ya aplicadas.

- PostGIS: si no está instalada, se crea en `extensions`. Si ya existe (en
  `extensions`, `gis` o `public`), se reutiliza `pg_extension.extnamespace` y
  `geom` se declara con el tipo calificado (`schema.geometry(Polygon)`). El
  `Down` no borra la extensión.
- `usuarios.inmobiliaria_id` queda nullable, sin CHECK: no hay mapeo de
  usuarios `inmobiliaria` existentes a agencias. La restricción
  (inmobiliaria_id NOT NULL iff rol = inmobiliaria) va en una migración
  posterior, después del backfill.
- `reservas.estado_actual` y `ventas.estado_actual` solo cambian insertando
  en el historial. Un trigger bloquea el UPDATE directo, las filas del
  historial rechazan UPDATE, DELETE y TRUNCATE, y el alta crea la primera
  fila `activa`.

Cada archivo debe tener una sección `Up` y una sección `Down`:

```sql
-- +goose Up
CREATE TABLE lots (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL
);

-- +goose Down
DROP TABLE lots;
```

Los marcadores `-- +goose Up` y `-- +goose Down` son obligatorios.

## Crear una migración

Desde la raíz, con la conexión traída de Doppler. Se usan las variables
`GOOSE_DRIVER`/`GOOSE_DBSTRING` en vez de pasar la connection string como
argumento: un argumento de línea de comandos queda visible para cualquier
proceso de la máquina (Task Manager, `Get-Process`), mientras que una
variable de entorno del proceso hijo no.

```powershell
$env:GOOSE_DRIVER = "postgres"
$env:GOOSE_DBSTRING = doppler secrets get DATABASE_URL --plain
go run github.com/pressly/goose/v3/cmd/goose@v3.27.3 -dir migrations create create_lots_table sql
```

Usar nombres descriptivos y una migración por cambio coherente. Goose genera
un archivo con numeración; no renombrarlo después de aplicarlo.

## Aplicar migraciones

La forma normal de desarrollo es:

```powershell
doppler run -- docker compose up --build
```

Compose ejecuta el servicio `migrate` contra Supabase y recién después inicia
el backend.

Para ejecutar únicamente el servicio de migraciones:

```powershell
doppler run -- docker compose up migrate
```

Para consultar el estado desde el host (con `$env:GOOSE_DRIVER`/
`$env:GOOSE_DBSTRING` ya resueltas como en el paso anterior):

```powershell
go run github.com/pressly/goose/v3/cmd/goose@v3.27.3 -dir migrations status
go run github.com/pressly/goose/v3/cmd/goose@v3.27.3 -dir migrations version
```

## Avanzar o retroceder manualmente

Aplicar una sola migración:

```powershell
go run github.com/pressly/goose/v3/cmd/goose@v3.27.3 -dir migrations up-by-one
```

Retroceder la última migración:

```powershell
go run github.com/pressly/goose/v3/cmd/goose@v3.27.3 -dir migrations down
```

`down` corre contra la base compartida de Supabase, no una base local
descartable: no usarlo sin confirmar el impacto con el equipo.

## Reglas del esquema

- Nunca modificar una migración que ya fue aplicada en algún ambiente.
- Crear una nueva migración para cada cambio posterior.
- Mantener `Up` y `Down` coherentes.
- Preferir cambios compatibles hacia atrás cuando una API y una base se
  actualizan en momentos distintos.
- No poner migraciones automáticas dentro de cada instancia del backend.
- Usar `-- +goose NO TRANSACTION` solo cuando PostgreSQL no permita ejecutar la
  operación dentro de una transacción.
- Revisar manualmente las migraciones destructivas antes de aplicarlas.
