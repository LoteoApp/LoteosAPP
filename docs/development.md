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

La autenticación no es un servicio local: el backend y el frontend hablan
con el proyecto de Supabase configurado en el `.env` de la raíz.

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
| `db` | `localhost:5432` | PostgreSQL de la aplicación |

Endpoints operativos del backend:

- `GET /healthz`: confirma que el proceso HTTP está activo.
- `GET /readyz`: confirma que el backend puede conectarse con PostgreSQL.
- `GET /api/v1/system`: devuelve el diagnóstico del backend y la base durante
  el desarrollo.
- `POST /api/v1/usuarios` (requiere rol `administrador`): da de alta un
  usuario nuevo en Supabase Auth y en Postgres, devuelve una contraseña
  temporal de un solo uso.
- `PATCH /api/v1/usuarios/me` (cualquier usuario autenticado): completa el
  propio perfil (nombre y apellido).

Si un puerto está ocupado, se puede cambiar sin editar Compose:

```powershell
$env:BACKEND_PORT = "8081"
$env:FRONTEND_PORT = "5174"
docker compose up --build
```

El puerto interno del backend continúa siendo `8080`; solo cambia el puerto
publicado en el host.

## Autenticación (Supabase Auth)

Backend y frontend están enteramente sobre Supabase Auth. No hay servicio de
identidad en Compose: ambos hablan con el proyecto de Supabase configurado en
el `.env` de la raíz.

### Backend

`internal/infrastructure/auth/supabase` valida cada Bearer token contra el
JWKS que publica el proyecto (`SUPABASE_URL`) y lee el rol de dominio de
`app_metadata.role`. El alta de usuarios usa la Admin REST API con la
`service_role` key.

El proceso no arranca si falta `SUPABASE_URL` o `SUPABASE_SERVICE_ROLE_KEY`:
es preferible fallar en el arranque a servir una API con la verificación de
tokens mal configurada.

### Frontend

`AppAuthProvider`
(`VITE_SUPABASE_URL`/`VITE_SUPABASE_ANON_KEY`) publica la sesión en un
`AuthContext`, y `RequireAuth`, `AuthStatus` y `UserMenu` la consumen desde
`features/auth/hooks/use-auth.ts`. Como Supabase no tiene pantalla de login
hosteada, el ingreso es un formulario propio de email y contraseña en `/login`;
`RequireAuth` manda ahí a quien no tenga sesión y recuerda la ruta pedida para
volver después del ingreso.

### Crear un usuario de Supabase para probar el login

Dashboard de Supabase → **Authentication > Users** → **Add user > Create new
user**, con **Auto Confirm User** activado. Ese usuario ya puede ingresar por
`/login`. El alta desde el backend (`AdminClient.CreateUser`) genera además una
contraseña temporal y guarda el rol de dominio en `app_metadata.role`.

## Proyecto de Supabase

Épica [#100](https://github.com/LoteoApp/LoteosAPP/issues/100). Queda
pendiente documentar el mapeo de roles de dominio
([#105](https://github.com/LoteoApp/LoteosAPP/issues/105)).

- **Proyecto**: `https://iahqjtpzkntzxoiykhjg.supabase.co` (entorno de
  desarrollo), creado con una cuenta Gmail dedicada al proyecto.
- **Claves**:
  - `anon`/`publishable`: pensada para el cliente, no es secreta. Se guarda
    igual en `.env` local (gitignorado) en vez de hardcodearla en el repo.
  - `service_role`: bypassea RLS, es secreta. Se obtiene manualmente desde
    el dashboard de Supabase (**Project Settings > API keys**) — no se expone
    vía MCP a propósito. Nunca se commitea ni se pega en docs/chat; solo vive
    en el `.env` local de quien la necesite.
- **Dónde se guardan**: `.env` en la raíz del repo (gitignorado, ver
  `.env.example` para las claves esperadas). Para CI/producción, cuando
  aplique, van como secrets de GitHub Actions (ver
  [ci.md](ci.md#secretos-y-credenciales)), no en `compose.yaml`.
- **Frontend fuera de Compose**: el `.env` de la raíz alcanza para
  `docker compose up`, que mapea `SUPABASE_URL`/`SUPABASE_ANON_KEY` a las
  `VITE_*` que lee Vite. Corriendo `pnpm --filter @loteos/frontend dev` por
  separado ese mapeo no existe, así que hace falta un `apps/frontend/.env`
  (también gitignorado) con los nombres ya prefijados:

  ```text
  VITE_SUPABASE_URL=...
  VITE_SUPABASE_ANON_KEY=...
  ```

  Sin `VITE_SUPABASE_ANON_KEY` el cliente arranca con la clave vacía y todo
  intento de login falla.
- **Acceso al proyecto**: invitar como miembro de la organización de
  Supabase en vez de compartir la contraseña de la cuenta Gmail.

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

El backend necesita el proyecto de Supabase para validar tokens y para dar de
alta usuarios por la Admin REST API. Ninguna de las dos tiene default: si
falta alguna, el proceso falla al arrancar.

```text
SUPABASE_URL=https://<proyecto>.supabase.co
SUPABASE_SERVICE_ROLE_KEY=sb_secret_...
```

La `service_role` key bypassea RLS y habilita la administración completa de
usuarios: va únicamente en el `.env` local (gitignorado) o en un secret de
CI, nunca en `compose.yaml` ni en la documentación.

El frontend, al correr en el navegador, necesita la URL y la clave pública
(publishable) del proyecto de Supabase:

```text
VITE_SUPABASE_URL=https://<proyecto>.supabase.co
VITE_SUPABASE_ANON_KEY=sb_publishable_...
```

Usar la clave `sb_publishable_...` (Project Settings → API Keys), no la
legacy `anon` en formato JWT: la legacy comparte el JWT secret del proyecto
con la `service_role`, así que rotarla obliga a rotar también la
`service_role`; la publishable rota de forma independiente. `VITE_SUPABASE_ANON_KEY`
es obligatoria en un build de producción (`env.ts` tira error si falta).
Estos valores no tienen un default de Compose (dependen del proyecto de
Supabase de cada entorno); se toman de `SUPABASE_URL`/`SUPABASE_ANON_KEY` en
el `.env` local (épica #100).

El backend recibe internamente esta conexión a PostgreSQL:

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
