# Desarrollo local

## Requisitos

- Docker Desktop con Docker Compose.
- Node.js 20.19+ y pnpm 10+.
- Go 1.26+.

El frontend usa React 19, Vite 8, TypeScript 7, Tailwind CSS 4 y shadcn/ui
(componentes en `apps/frontend/src/shared/ui`). El backend usa Go 1.26,
`pgx/v5/pgxpool` y Goose para migraciones.

## Arrancar todo con Docker

Desde la raíz del repositorio:

```powershell
docker compose up --build
```

La secuencia de inicio es:

1. `db` inicia PostgreSQL y espera su health check.
2. `migrate` aplica las migraciones pendientes y termina con código 0.
3. `backend` inicia la API cuando la base y las migraciones están listas.
4. `frontend` inicia Vite cuando el backend está saludable.

Para ejecutar en segundo plano:

```powershell
docker compose up --build -d
```

Para detenerlo:

```powershell
docker compose down
```

`docker compose down` conserva el volumen `postgres_data`. Para eliminarlo y
reiniciar la base desde cero:

```powershell
docker compose down -v
docker compose up --build
```

El segundo comando es destructivo para los datos locales de PostgreSQL.

## Servicios y puertos

| Servicio | URL/puerto del host | Uso |
| --- | --- | --- |
| `frontend` | `http://localhost:5173` | Vite con recarga en caliente |
| `backend` | `http://localhost:8080` | API Go |
| `db` | `localhost:5432` | PostgreSQL |

Endpoints operativos del backend:

- `GET /healthz`: confirma que el proceso HTTP está activo.
- `GET /readyz`: confirma que el backend puede conectarse con PostgreSQL.
- `GET /api/v1/system`: devuelve el diagnóstico del backend y la base durante
  el desarrollo.

Si un puerto está ocupado, se puede cambiar sin editar Compose:

```powershell
$env:BACKEND_PORT = "8081"
$env:FRONTEND_PORT = "5174"
docker compose up --build
```

El puerto interno del backend continúa siendo `8080`; solo cambia el puerto
publicado en el host.

## Variables de entorno

Compose usa estos valores por defecto:

```text
POSTGRES_DB=loteosapp
POSTGRES_USER=loteosapp
POSTGRES_PASSWORD=loteosapp
POSTGRES_PORT=5432
BACKEND_PORT=8080
FRONTEND_PORT=5173
```

El backend recibe internamente esta conexión:

```text
postgres://loteosapp:loteosapp@db:5432/loteosapp?sslmode=disable
```

Desde el host se usa `localhost` en lugar de `db`:

```text
postgres://loteosapp:loteosapp@localhost:5432/loteosapp?sslmode=disable
```

## Trabajar sin Docker para frontend o backend

No se deben iniciar dos instancias del mismo servicio en el mismo puerto.

Para trabajar con el frontend en el host:

```powershell
pnpm install
pnpm dev
```

### HMR del frontend

Vite tiene HMR activado por defecto. Como el proyecto está montado desde
Windows dentro de un contenedor Docker, `apps/frontend/vite.config.ts` usa
polling para detectar cambios que no siempre llegan mediante eventos nativos
del sistema de archivos.

Al editar `apps/frontend/src/`, el navegador debe actualizarse sin reiniciar el
contenedor. El polling consume algo más de CPU; si el proyecto se mueve a un
entorno con eventos de archivos confiables, se puede quitar esa configuración.

Para trabajar con el backend en el host, primero debe estar PostgreSQL
disponible y debe configurarse `DATABASE_URL`:

```powershell
$env:DATABASE_URL = "postgres://loteosapp:loteosapp@localhost:5432/loteosapp?sslmode=disable"
cd apps/backend
go run ./cmd/server
```

El contenedor del frontend tiene recarga en caliente. El contenedor del
backend monta el código fuente, pero ejecuta `go run` sin un watcher; después
de cambios en Go se debe reiniciar:

```powershell
docker compose restart backend
```

## Verificación

Desde la raíz:

```powershell
docker compose config
pnpm --filter @loteos/frontend typecheck
pnpm --filter @loteos/frontend lint
pnpm --filter @loteos/frontend build
```

Desde `apps/backend`:

```powershell
go test ./...
go vet ./...
```
