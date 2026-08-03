# PostgreSQL y migraciones

## Conexión recomendada desde Go

Para la API se usa `github.com/jackc/pgx/v5/pgxpool`.

Es la opción recomendada porque:

- Está diseñada para PostgreSQL y sus características específicas.
- Administra un pool de conexiones seguro para múltiples requests concurrentes.
- Permite configurar límites, tiempos de vida, conexiones mínimas y health checks.
- Mantiene las queries parametrizadas con placeholders `$1`, `$2`, etc.

La conexión del backend se configura en
`apps/backend/internal/platform/postgres/pool.go` y se construye desde
`apps/backend/internal/app/app.go`. Al iniciar:

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

## Endpoint de diagnóstico

El backend expone `GET /api/v1/system` para validar el entorno durante el
desarrollo. Devuelve, sin incluir la contraseña ni la URL completa de conexión:

- Estado de conexión.
- Versión completa de PostgreSQL.
- Nombre de la base y usuario conectado.
- Dirección y puerto del servidor.
- Hora actual de PostgreSQL.
- Conexiones máximas, totales, adquiridas, libres, creadas y cerradas del pool.

El componente
`apps/frontend/src/features/system-status/components/DatabaseStatus.tsx`
consulta este endpoint a través de la capa `api` de su feature y muestra la
información en la pantalla principal. Este endpoint es diagnóstico de
desarrollo y debe revisarse antes de exponerlo en producción.

## Herramienta de migraciones

Se usa [Goose](https://github.com/pressly/goose) con archivos SQL versionados.
El backend usa `pgxpool`; el comando de migraciones usa `database/sql` con el
driver `github.com/jackc/pgx/v5/stdlib`, porque Goose trabaja con `*sql.DB`.

Las migraciones están en:

```text
migrations/
└── 00001_init.sql
```

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

Desde la raíz, apuntando a PostgreSQL publicado por Docker:

```powershell
$env:DATABASE_URL = "postgres://loteosapp:loteosapp@localhost:5432/loteosapp?sslmode=disable"
go run github.com/pressly/goose/v3/cmd/goose@v3.27.3 -dir migrations postgres $env:DATABASE_URL create create_lots_table sql
```

Usar nombres descriptivos y una migración por cambio coherente. Goose genera
un archivo con numeración; no renombrarlo después de aplicarlo.

## Aplicar migraciones

La forma normal de desarrollo es:

```powershell
docker compose up --build
```

Compose espera que PostgreSQL esté saludable, ejecuta el servicio `migrate` y
recién después inicia el backend.

Para ejecutar únicamente el servicio de migraciones:

```powershell
docker compose up migrate
```

Para consultar el estado desde el host:

```powershell
$env:DATABASE_URL = "postgres://loteosapp:loteosapp@localhost:5432/loteosapp?sslmode=disable"
go run github.com/pressly/goose/v3/cmd/goose@v3.27.3 -dir migrations postgres $env:DATABASE_URL status
go run github.com/pressly/goose/v3/cmd/goose@v3.27.3 -dir migrations postgres $env:DATABASE_URL version
```

## Avanzar o retroceder manualmente

Aplicar una sola migración:

```powershell
go run github.com/pressly/goose/v3/cmd/goose@v3.27.3 -dir migrations postgres $env:DATABASE_URL up-by-one
```

Retroceder la última migración:

```powershell
go run github.com/pressly/goose/v3/cmd/goose@v3.27.3 -dir migrations postgres $env:DATABASE_URL down
```

No usar `down` en una base compartida sin confirmar el impacto. Para resetear
la base local completa:

```powershell
docker compose down -v
docker compose up --build
```

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
