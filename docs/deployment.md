# Despliegue en Dokploy

Ambas apps se despliegan en Dokploy como **Applications** independientes
(build type `Dockerfile`), no como un stack de Docker Compose: cada una tiene
su propio dominio, sus propias variables y su propio ciclo de release. La
base de datos y la autenticación siguen siendo el proyecto de Supabase
([database.md](database.md)); Dokploy nunca corre Postgres.

## Imágenes

| Stage | Imagen | Por qué |
| --- | --- | --- |
| Build backend | `golang:1.26-alpine` | Misma versión que `apps/backend/go.mod` (`go 1.26.6`); alpine solo para compilar, no llega a producción. |
| Runtime backend | `gcr.io/distroless/static-debian12:nonroot` | Sin shell, sin gestor de paquetes, corre como `nonroot` (uid 65532) por defecto. Incluye `ca-certificates`, necesario para TLS contra Supabase (`sslmode=require`) y R2. El binario se compila `CGO_ENABLED=0` (estático, verificado), así que no hace falta libc en la imagen final. |
| Build frontend | `node:24-alpine` | Misma mayor que `engines.node` (`>=20.19.0`) en `apps/frontend/package.json`. |
| Runtime frontend | `nginxinc/nginx-unprivileged:1.29-alpine` | Variante oficial de nginx que corre como usuario no root desde el arranque (sin necesidad de `USER`/`chown` manuales) y escucha en `8080`, un puerto no privilegiado. |

Ninguna imagen usa `latest`: fijar tags concretos evita que un rebuild
cambie de versión sin aviso.

## Backend (`apps/backend/Dockerfile`)

Tres stages: `builder`, `migrate` y `server` (default, por ser el último
`FROM`). El contexto de build es la **raíz del repo**, no `apps/backend`,
porque el stage `migrate` necesita copiar `migrations/`, que vive fuera de
`apps/backend`.

Configuración en Dokploy (Application → Build):

- **Build type**: `Dockerfile`
- **Dockerfile Path**: `apps/backend/Dockerfile`
- **Docker Context Path**: `.`
- **Docker Build Stage**: vacío (usa `server`)

### Variables de entorno (runtime, no build args)

El backend no necesita nada en build time — todo se lee en el arranque
(`environments.LoadServer`, ver [secrets.md](secrets.md)):

| Variable | Origen |
| --- | --- |
| `DATABASE_URL` | Doppler, config `prd`. Pooler de Supabase en modo session (puerto `5432`), no el modo transaction. |
| `PORT` | `8080` (coincide con el `EXPOSE` de la imagen). |
| `FRONTEND_ORIGIN` | Dominio de producción del frontend, para CORS. Debe ser exacto (incluye esquema, sin barra final). |
| `SUPABASE_URL`, `SUPABASE_SERVICE_ROLE_KEY` | Doppler, config `prd`. |
| `CLOUDFLARE_R2_ENDPOINT`, `CLOUDFLARE_R2_BUCKET_NAME`, `CLOUDFLARE_R2_ACCESS_KEY_ID`, `CLOUDFLARE_R2_SECRET_ACCESS_KEY` | Doppler, config `prd`. El bucket debe ser uno de producción (`loteos-files-prd`), separado del de desarrollo — ver "Bucket R2 de producción" más abajo. |

Cargar estas variables como **Environment Variables** de la Application, no
como Build Arguments: el backend no las necesita durante el build, y
guardarlas ahí evita que terminen en el historial de capas de la imagen.

### Health check

El backend no exige auth en `GET /health` (agregado en este cambio —
[route.go](../apps/backend/internal/infrastructure/delivery/webapp/route/route.go)
— justo porque todas las demás rutas la exigen y Dokploy necesita algo que
pueda probar sin credenciales).

Ninguna imagen `distroless` (ni siquiera `base-debian12`) trae `wget` ni
`curl` — solo las variantes `:debug` agregan un shell busybox limitado, y no
las usamos en producción porque pierden buena parte de las garantías de
`distroless` (superficie de ataque mínima, sin shell). En vez de depender de
una herramienta externa dentro del contenedor, el propio binario del server
expone un subcomando `healthcheck`
([cmd/server/main.go](../apps/backend/cmd/server/main.go)) que hace un
`GET http://127.0.0.1:$PORT/health` y sale con código `0`/`1` según el
resultado. El `Dockerfile` ya declara el `HEALTHCHECK` con ese `CMD`:

```dockerfile
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
  CMD ["/app/server", "healthcheck"]
```

Dokploy respeta el `HEALTHCHECK` de la imagen sin configuración adicional en
**Advanced → Swarm Settings**; solo hace falta tocar algo ahí si se quiere
sobrescribir el intervalo/reintentos definidos en el Dockerfile.

## Frontend (`apps/frontend/Dockerfile`)

Configuración en Dokploy (Application → Build):

- **Build type**: `Dockerfile`
- **Dockerfile Path**: `apps/frontend/Dockerfile`
- **Docker Context Path**: `.`

### Variables — build-time, no runtime

Vite hornea las variables `VITE_*` en el bundle en el momento del build
([env-and-mode](https://vite.dev/guide/env-and-mode)): cambiarlas después
requiere un rebuild, no alcanza con reiniciar el contenedor. Van como
**Build Time Arguments** en Dokploy, no como Environment Variables:

| Build arg | Valor |
| --- | --- |
| `VITE_API_URL` | URL pública del backend de producción. |
| `VITE_SUPABASE_URL` | Igual que `SUPABASE_URL` del backend. |
| `VITE_SUPABASE_ANON_KEY` | La `anon key` de Supabase — pública por diseño ([database.md](database.md#exposición-por-la-data-api-y-rls)), no hace falta Build-time Secret para esta. |

No usar **Build-time Secrets** para estas tres: los secrets de Docker no
persisten en la imagen final, pero acá el objetivo es exactamente lo
opuesto — necesitan quedar en el bundle JS para que el navegador las use.
`SUPABASE_SERVICE_ROLE_KEY` nunca debe tocar el frontend, en ningún build
arg ni secret: viaja únicamente al backend.

### Puerto y dominio

La imagen escucha en `8080` (puerto no privilegiado de
`nginx-unprivileged`). Al crear el dominio en Dokploy, `Container Port` debe
ser `8080`, no `80`.

## Migraciones

Regla existente del repo ([database.md](database.md#reglas-del-esquema)): no
correr migraciones automáticamente en cada instancia del backend. Eso no
cambia con Dokploy — el `server` stage no las ejecuta al arrancar.

Camino recomendado, el mismo que ya usa el equipo en desarrollo, apuntado a
`prd`: correr `cmd/migrate` a mano, con `DATABASE_URL` de Doppler, antes de
desplegar el backend:

```powershell
doppler run --config prd -- go run ./cmd/migrate
```

Alternativa en contenedor, con el stage `migrate` de la misma imagen (útil
si no hay Go instalado en la máquina que despliega):

```bash
docker build --target migrate -t loteosapp-migrate -f apps/backend/Dockerfile .
docker run --rm -e DATABASE_URL="postgres://..." loteosapp-migrate
```

No configurar esto como un **Schedule Job** de Dokploy que corra en cada
deploy: correría sin coordinación con el orden real de los cambios y sin
que nadie revise el resultado antes de que el backend nuevo empiece a
recibir tráfico.

## Bucket R2 de producción

`docs/secrets.md` documenta un bucket por entorno. Antes de este deploy
solo existe `loteos-files-dev`. Crear `loteos-files-prd` siguiendo el mismo
procedimiento (`wrangler r2 bucket create loteos-files-prd`, token propio
con permiso Object Read & Write acotado a ese bucket) y cargar sus
credenciales en la config `prd` de Doppler.

## CORS

`FRONTEND_ORIGIN` en el backend debe ser exactamente el origen del
frontend de producción (`https://dominio-frontend`, sin path ni barra
final). Un mismatch no rompe el build ni el arranque — se manifiesta recién
como error de CORS en el browser al primer request real.

## Seguridad

Lo que aplica puntualmente a estas dos apps, a partir de la [guía de
production hardening de
Dokploy](https://github.com/dokploy/website/blob/main/apps/docs/content/docs/core/guides/production-hardening.mdx):

- **Ambas imágenes corren sin root**: `distroless:nonroot` en el backend,
  `nginx-unprivileged` en el frontend. Ninguna necesita `USER` adicional ni
  puertos privilegiados.
- **`security_opt: no-new-privileges:true`**: agregarlo en Advanced Settings
  de cada Application. Evita que un proceso dentro del contenedor escale
  privilegios aunque el runtime ya sea non-root.
- **Secrets nunca como build args**: los `CLOUDFLARE_R2_*` y
  `SUPABASE_SERVICE_ROLE_KEY` van como Environment Variables del backend
  (runtime), nunca como Build Time Arguments — un build arg queda en el
  historial de capas de la imagen aunque no se use en el resultado final.
- **HTTPS + redirect forzado** en el dominio de ambas apps (Traefik /
  Let's Encrypt, configuración estándar de Dokploy), y **HSTS** en el
  dominio del frontend.
- **`FRONTEND_ORIGIN` exacto**: es el único control de CORS del backend; no
  usar un wildcard ni un valor más amplio que el dominio real.
- **Bucket R2 separado por entorno** (sección anterior): una credencial de
  `prd` filtrada no debe alcanzar los archivos de `dev`.
- **No exponer el puerto del backend directamente** (`Ports` en Advanced
  Settings) salvo necesidad puntual de depurar por IP — el dominio ya lo
  enruta vía Traefik.
- **Rol de base de datos administrativo** ([database.md](database.md#conexión-a-supabase)):
  sigue pendiente separar un rol de aplicación de permisos mínimos del rol
  que corre las migraciones. No es específico de Dokploy, pero pasar a
  producción es el momento de resolverlo antes de manejar datos reales.
- Lo que queda fuera del alcance de este repo porque vive en la instancia de
  Dokploy o en la infraestructura del servidor (2FA en la cuenta, firewall
  del host, rotación del socket de Docker, backups cifrados, etc.) está en
  el checklist completo de la guía enlazada arriba — revisarlo una vez al
  configurar el servidor, no por app.

## Verificación local

Los tres builds (`server`, `migrate`, frontend) se probaron con Docker
Desktop:

- Los tres `docker build` terminan sin errores.
- `docker run` de `server` y `migrate` sin variables de entorno falla rápido
  con el mensaje de validación esperado (`DATABASE_URL, SUPABASE_URL, ...
  must be set` / `DATABASE_URL must be set`) en vez de un panic o un error
  de runtime — confirma que el binario estático corre bien en
  `distroless:nonroot` y que no llegó a intentar nada sin configuración.
- El contenedor de nginx responde `200` en `/` y en una ruta profunda
  inexistente (fallback SPA vía `try_files`), con los headers de seguridad
  del `nginx.conf` presentes, corriendo como `uid=101(nginx)` (no root).

No se probó el arranque completo del backend contra una base real (necesita
credenciales de Supabase/R2 de verdad). El subcomando `healthcheck` sí tiene
cobertura de tests (`cmd/server/main_test.go`), pero conviene verificar una
vez en Dokploy que el `HEALTHCHECK` del Dockerfile efectivamente marca el
contenedor como sano antes de depender de él en producción.

Repetir los builds después de cualquier cambio a los Dockerfiles:

```powershell
docker build -f apps/backend/Dockerfile -t loteosapp-backend .
docker build -f apps/backend/Dockerfile --target migrate -t loteosapp-migrate .
docker build -f apps/frontend/Dockerfile `
  --build-arg VITE_API_URL=https://api.ejemplo.com `
  --build-arg VITE_SUPABASE_URL=https://xxx.supabase.co `
  --build-arg VITE_SUPABASE_ANON_KEY=eyJ... `
  -t loteosapp-frontend .
```
