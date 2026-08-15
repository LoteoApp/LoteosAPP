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

1. `db` y `keycloak-db` inician PostgreSQL (bases separadas) y esperan su
   health check.
2. `migrate` aplica las migraciones pendientes y termina con código 0.
3. `keycloak` inicia e importa el realm `loteosapp` desde
   `keycloak/realm-loteosapp.json`.
4. `backend` inicia la API cuando la base, las migraciones y Keycloak están
   listos.
5. `frontend` inicia Vite cuando el backend está saludable.

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
| `keycloak` | `http://localhost:8081` | Consola admin y endpoints OIDC |
| `keycloak-db` | interno (sin puerto publicado) | PostgreSQL dedicado a Keycloak |

Endpoints operativos del backend:

- `GET /healthz`: confirma que el proceso HTTP está activo.
- `GET /readyz`: confirma que el backend puede conectarse con PostgreSQL.
- `GET /api/v1/system`: devuelve el diagnóstico del backend y la base durante
  el desarrollo.
- `POST /api/v1/usuarios` (requiere rol `administrador`): da de alta un
  usuario nuevo en Keycloak y en Postgres, devuelve una contraseña temporal
  de un solo uso.
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

## Keycloak (autenticación)

`docker compose up --build` levanta Keycloak con un realm ya configurado para
la aplicación, sin pasos manuales.

### Qué trae el realm

Al arrancar, el contenedor `keycloak` importa
`keycloak/realm-loteosapp.json` (flag `--import-realm`) con:

- **Realm** `loteosapp`, con el login en español
  (`internationalizationEnabled`, `defaultLocale: es`).
- **Clients**:
  - `loteosapp-frontend`: público, con PKCE (S256), pensado para el flujo
    Authorization Code de la SPA (`redirectUris: http://localhost:5173/*`).
  - `loteosapp-backend`: confidencial (`client-secret`,
    `loteosapp-backend-dev-secret` — solo para desarrollo local, mismo
    criterio que el resto de las credenciales de `compose.yaml`), con
    service accounts habilitado para llamadas servidor-a-servidor.
- **Realm roles**, uno por cada rol de dominio documentado en
  [domain.md](domain.md#usuarios-y-roles): `administrador`, `administrativo`,
  `agrimensor`, `escribano`, `inmobiliaria`.

El import usa la estrategia `IGNORE_EXISTING`: si el realm ya existe (por
ejemplo, porque el volumen `keycloak_postgres_data` es de una corrida
anterior), el archivo se ignora y no pisa cambios hechos a mano desde la
consola. Para que una edición de `realm-loteosapp.json` se aplique sobre un
entorno que ya lo tenía importado, hay que borrar ese volumen y volver a
levantar:

```powershell
docker compose down
docker volume rm loteosapp_keycloak_postgres_data
docker compose up --build
```

Esto es destructivo: borra también los usuarios y cualquier cambio hecho a
mano en ese realm.

### Consola de administración

`http://localhost:8081`, realm `master`, usuario/contraseña
`KEYCLOAK_ADMIN_USER`/`KEYCLOAK_ADMIN_PASSWORD` (por defecto `admin`/`admin`).
Desde ahí hay que cambiar al realm `loteosapp` (selector arriba a la
izquierda) para ver los clients y roles importados.

### Crear un usuario de Keycloak para probar la API

El login de la app ya no pasa por Keycloak (ver [Backend y
frontend](#backend-y-frontend) más abajo). Un usuario del realm sirve
únicamente para obtener un token con el que llamar al backend a mano, hasta
que [#103](https://github.com/LoteoApp/LoteosAPP/issues/103) corte la
validación a Supabase. El realm no trae usuarios cargados:

1. Consola admin → realm `loteosapp` → **Users** → **Add user**.
2. Cargar username/email → **Create**.
3. Pestaña **Credentials** → **Set password**. Si se deja el toggle
   **Temporary** activado, Keycloak va a pedir cambiarla en el primer login
   (funciona igual, es un paso extra).
4. Opcional, pestaña **Role mapping** → asignar uno de los roles de dominio.

### Backend y frontend

El backend todavía valida los tokens que emite Keycloak (JWKS del realm) vía
`KEYCLOAK_ISSUER`/`KEYCLOAK_AUDIENCE`; ver [Variables de
entorno](#variables-de-entorno) más abajo. El corte del backend a Supabase es
la tarea [#103](https://github.com/LoteoApp/LoteosAPP/issues/103).

El frontend ya está enteramente sobre Supabase: `AppAuthProvider`
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

## Supabase (en migración desde Keycloak)

Épica [#100](https://github.com/LoteoApp/LoteosAPP/issues/100). El frontend ya
usa Supabase de punta a punta y el driver de backend existe
(`internal/infrastructure/auth/supabase`) pero todavía no está cableado;
`compose.yaml` y el backend siguen levantando Keycloak (tareas
[#102](https://github.com/LoteoApp/LoteosAPP/issues/102),
[#103](https://github.com/LoteoApp/LoteosAPP/issues/103),
[#105](https://github.com/LoteoApp/LoteosAPP/issues/105) y
[#108](https://github.com/LoteoApp/LoteosAPP/issues/108)).

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
KEYCLOAK_PORT=8081
KEYCLOAK_REALM=loteosapp
KEYCLOAK_BACKEND_CLIENT=loteosapp-backend
KEYCLOAK_FRONTEND_CLIENT=loteosapp-frontend
KEYCLOAK_ADMIN_USER=admin
KEYCLOAK_ADMIN_PASSWORD=admin
KEYCLOAK_DB=keycloak
KEYCLOAK_DB_USER=keycloak
KEYCLOAK_DB_PASSWORD=keycloak
```

El backend recibe internamente la URL de Keycloak (resuelve `keycloak` por
nombre de servicio dentro de la red de Compose) para validar tokens, y
necesita además llamar a la Admin REST API de Keycloak (alta de usuarios),
autenticándose con las credenciales del client `loteosapp-backend`:

```text
KEYCLOAK_ISSUER=http://keycloak:8080/realms/loteosapp
KEYCLOAK_JWKS_BASE_URL=http://keycloak:8080/realms/loteosapp
KEYCLOAK_AUDIENCE=loteosapp-backend
KEYCLOAK_BASE_URL=http://keycloak:8080
KEYCLOAK_REALM=loteosapp
KEYCLOAK_CLIENT_SECRET=loteosapp-backend-dev-secret
```

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
