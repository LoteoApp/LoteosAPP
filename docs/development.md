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

1. `migrate` aplica las migraciones pendientes contra Supabase y termina con
   código 0.
2. `backend` inicia la API cuando las migraciones están listas.
3. `frontend` inicia Vite cuando el proceso del backend arrancó (sin esperar
   ninguna verificación de salud; ver `docker compose logs backend` si el
   backend no llega a conectarse a la base).

Ni la autenticación ni la base de datos de la aplicación son servicios
locales: el backend, `migrate` y el frontend hablan con el proyecto de
Supabase configurado en el `.env` de la raíz ([#127](https://github.com/LoteoApp/LoteosAPP/issues/127)).
El servicio `db` de Compose sigue arriba pero ya no lo usa nadie; se elimina
en [#128](https://github.com/LoteoApp/LoteosAPP/issues/128).

Para ejecutar en segundo plano:

```powershell
docker compose up --build -d
```

Para detenerlo:

```powershell
docker compose down
```

```powershell
docker compose down -v
```

Elimina también el volumen `postgres_data` del servicio `db`, que ya no
respalda a `backend` ni a `migrate` ([#127](https://github.com/LoteoApp/LoteosAPP/issues/127))
y se retira en [#128](https://github.com/LoteoApp/LoteosAPP/issues/128): hoy
no borra ni resetea la base real de la aplicación, que vive en Supabase y es
compartida por todo el equipo. No hay equivalente local de "resetear la base
desde cero"; usar `goose down`/`goose up` (ver
[database.md](database.md#avanzar-o-retroceder-manualmente)) contra Supabase,
con cuidado, para retroceder cambios.

## Servicios y puertos

| Servicio | URL/puerto del host | Uso |
| --- | --- | --- |
| `frontend` | `http://localhost:5173` | Vite con recarga en caliente |
| `backend` | `http://localhost:8080` | API Go |
| `db` | `localhost:5432` | En desuso; se elimina en [#128](https://github.com/LoteoApp/LoteosAPP/issues/128) |

Endpoints operativos del backend:

- `POST /api/v1/usuarios` (requiere rol `administrador`): da de alta un
  usuario nuevo en Supabase Auth y en Postgres, devuelve una contraseña
  temporal de un solo uso.
- `PATCH /api/v1/usuarios/me` (cualquier usuario autenticado): completa el
  propio perfil (nombre y apellido).

### Content-Security-Policy del frontend

Un plugin de Vite (`apps/frontend/vite-plugins/content-security-policy.ts`)
inyecta un `<meta http-equiv="Content-Security-Policy">` en build time —
mitigación en capas contra XSS, dado que `supabase-js` guarda la sesión en
`localStorage` ([#112](https://github.com/LoteoApp/LoteosAPP/issues/112)).
Detalle y justificación de cada directiva en
[architecture.md](architecture.md#seguridad-del-frontend).

`connect-src` cubre dos orígenes, tomados de las mismas variables que usa el
resto del frontend:

- `VITE_SUPABASE_URL` (origen exacto, sin wildcard — un wildcard
  `*.supabase.co` dejaría pasar cualquier proyecto Supabase ajeno).
- `VITE_API_URL` — el mismo origen del backend documentado en la tabla de
  arriba.

Si ninguna de las dos está seteada, el plugin usa los mismos defaults que
`shared/config/env.ts` (`shared/config/env-defaults.ts`), así que nunca
queda un placeholder sin resolver ni una política rota por falta de
variable — a diferencia de escribir la política a mano en `index.html` con
sustitución `%VAR%` de Vite.

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
BACKEND_PORT=8080
FRONTEND_PORT=5173
```

`POSTGRES_DB`/`POSTGRES_USER`/`POSTGRES_PASSWORD`/`POSTGRES_PORT` solo los usa
el servicio `db`, que ya no respalda a `backend` ni a `migrate` y se elimina
en [#128](https://github.com/LoteoApp/LoteosAPP/issues/128).

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

El backend y `migrate` necesitan la base de Supabase
([#127](https://github.com/LoteoApp/LoteosAPP/issues/127)). Ver
[database.md § Conexión a Supabase](database.md#conexión-a-supabase) para el
detalle del pooler y por qué el modo, el `sslmode` y el tamaño del pool
importan.

```text
DATABASE_URL=postgres://postgres.<project-ref>:<password>@aws-<region>.pooler.supabase.com:5432/postgres?sslmode=require
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

Para trabajar con el backend en el host, con `DATABASE_URL` (y las
`SUPABASE_*`) inyectadas por Doppler:

```powershell
cd apps/backend
doppler run -- go run ./cmd/server
```

Como la base y la autenticación son de Supabase, todo el entorno puede correr
sin Docker: el backend con el comando de arriba y el frontend con
`pnpm --filter @loteos/frontend dev`. El frontend no necesita Doppler porque
lee `apps/frontend/.env` (Vite solo expone variables con prefijo `VITE_`, y en
Doppler los secrets están sin ese prefijo).

Fuera de Compose nadie aplica las migraciones: no hay un servicio `migrate`
que corra antes del backend, así que las pendientes se aplican a mano. El
default de `MIGRATIONS_DIR` es `migrations` relativo al directorio actual, que
desde `apps/backend` no resuelve:

```powershell
cd apps/backend
$env:MIGRATIONS_DIR = "$(Resolve-Path ../../migrations)"
doppler run -- go run ./cmd/migrate
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
