# Arquitectura y estructura

## Objetivo

LoteosAPP sigue una arquitectura modular por funcionalidad. Se aplican los
principios útiles de Clean Architecture —límites claros, dependencias hacia el
negocio y adaptadores reemplazables— sin crear capas o abstracciones que todavía
no aporten valor.

Las prioridades son:

- claridad del código;
- facilidad para localizar y modificar una funcionalidad;
- reglas del negocio independientes de HTTP, PostgreSQL y React;
- dependencias explícitas y simples de probar;
- crecimiento incremental, sin infraestructura anticipada.

## Vista general

```mermaid
flowchart LR
    frontend["Frontend React/Vite"] -->|HTTP| backend["Backend Go"]
    backend -->|pgxpool| db["PostgreSQL (Supabase)"]
    migrate["Goose migrate"] -->|aplica SQL| db
    migrate -->|service_completed_successfully| backend
    backend -->|healthcheck| frontend
```

## Backend

### Estructura objetivo

```text
apps/backend/
├── cmd/
│   ├── server/
│   │   └── main.go
│   └── migrate/
│       └── main.go
├── internal/
│   ├── app/
│   │   └── app.go
│   ├── business/
│   │   ├── domain/
│   │   │   └── system.go
│   │   ├── gateway/
│   │   │   ├── repository.go
│   │   │   └── gatewayfake/
│   │   │       └── repository.go
│   │   └── usecase/
│   │       ├── system/
│   │       │   └── get_system_info.go
│   │       └── users/
│   │           └── create_user.go
│   └── infrastructure/
│       ├── environments/
│       │   └── config.go
│       ├── auth/
│       │   └── supabase/
│       │       ├── admin.go
│       │       └── verifier.go
│       ├── repository/
│       │   └── postgres/
│       │       ├── pool.go
│       │       └── repository.go
│       └── delivery/
│           └── webapp/
│               ├── dependencies/
│               │   └── dependencies.go
│               ├── dto/
│               │   ├── system/
│               │   │   └── status.go
│               │   └── users/
│               │       └── create_user.go
│               ├── handler/
│               │   ├── live.go
│               │   ├── get_system_info.go
│               │   └── create_user.go
│               ├── middleware/
│               │   └── auth.go
│               ├── response/
│               │   └── json.go
│               ├── route/
│               │   └── route.go
│               └── server/
│                   ├── server.go
│                   └── cors.go
├── Dockerfile.dev
├── go.mod
└── go.sum
```

Esta estructura se adoptará incrementalmente. No es necesario crear directorios
vacíos antes de que exista una funcionalidad que los necesite.

### Responsabilidades

- `cmd/server`: inicia el proceso, recibe señales y delega la construcción de la
  aplicación. Debe contener muy poca lógica.
- `cmd/migrate`: ejecuta las migraciones y termina.
- `internal/app`: composition root y ciclo de vida del proceso. Al iniciar, lee
  la configuración (`environments`), pide a `dependencies` el grafo de
  objetos ya armado, registra las rutas y construye el `*http.Server`. Además
  sirve requests y apaga todo de forma ordenada ante una señal de cierre.
- `internal/infrastructure/delivery/webapp/dependencies`: contenedor de
  inyección de dependencias (IoC). Recibe lo que necesita desde `internal/app`
  (por ejemplo la cadena de conexión) y arma el grafo repositorio → servicio →
  handler, listo para usar. No conoce configuración, rutas ni el servidor
  HTTP; eso es responsabilidad de `internal/app`.
- `internal/business/domain`: entidades y tipos de valor del negocio, sin
  depender de HTTP ni de PostgreSQL.
- `internal/business/gateway`: contratos (interfaces) que el negocio necesita de
  sus adaptadores, como `Repository`. Los casos de uso dependen de estos
  contratos, nunca de una implementación concreta. `gateway/gatewayfake`
  contiene fakes de esos contratos para tests, ubicados junto a las
  interfaces que implementan en vez de duplicarse en cada paquete de
  `usecase`.
- `internal/business/usecase`: casos de uso que orquestan el dominio a través de
  los contratos de `gateway`, agrupados en subpaquetes por funcionalidad
  (`usecase/system`, `usecase/users`). Cada caso de uso es una interfaz de un
  solo método `Execute` junto con su implementación, definidas en el mismo
  archivo (por ejemplo `usecase/users/create_user.go`).
- `internal/infrastructure/environments`: carga de configuración desde
  variables de entorno.
- `internal/infrastructure/auth/supabase`: valida el JWT (Bearer token)
  emitido por Supabase Auth contra el JWKS del proyecto y extrae el `sub`,
  el email y el rol de dominio leído de `app_metadata.role`. También
  implementa `gateway.IdentityProvider` contra la Admin REST API de
  Supabase (alta/baja de usuarios), autenticándose con la `service_role`
  key. Los 5 roles de dominio (`administrador`, `administrativo`,
  `agrimensor`, `escribano`, `inmobiliaria`, ver `internal/business/domain`)
  son los mismos que tenía el realm de Keycloak: no hubo remapeo de nombres
  al migrar, solo cambio de transporte (`realm_access.roles` en el JWT de
  Keycloak → `app_metadata.role`, un único string, en el de Supabase).
- `internal/infrastructure/delivery/webapp/middleware`: adapta la
  validación de `auth/supabase` a un middleware HTTP; rechaza requests sin
  token válido y expone el llamador autenticado al resto de la request.
- `internal/infrastructure/repository/postgres`: implementa los contratos de
  persistencia (`gateway.Repository`) con `pgxpool` y SQL explícito, y expone
  la apertura y configuración del pool de conexiones.
- `internal/infrastructure/delivery/webapp/dto`: structs de request/response
  HTTP, agrupados por feature (`dto/system`, `dto/users`) igual que
  `usecase`. Cada subpaquete declara `package dto`; como el identificador de
  paquete no tiene que coincidir con el nombre del directorio, no choca con
  `usecase/system` ni `usecase/users` al importarse en el mismo archivo de
  `handler`.
- `internal/infrastructure/delivery/webapp/handler`: adapta requests HTTP a
  llamadas del caso de uso, decodificando y codificando los tipos de `dto`, y
  convierte el resultado en una respuesta. Cada ruta tiene su propio handler
  independiente, con un único caso de uso como dependencia (por ejemplo
  `CreateUserHandler` sólo conoce `users.CreateUser`); no hay un handler
  compartido que agrupe varias rutas. Un handler implementa `HTTPHandler`
  (`Handle(w, r) error`) en vez de escribir su propia respuesta de error: el
  error del caso de uso (o de `decodeJSON`) simplemente se retorna, y
  `Adapt(h, timeout)` lo convierte en `http.HandlerFunc` llamando a
  `response.WriteError`. `Adapt` también acota `r.Context()` al `timeout`
  antes de llamar a `Handle`, así que ningún handler arma su propio
  `context.WithTimeout`; usa `request.Context()` directo. Un handler sin
  ningún error posible (como `Live`) queda como función suelta, sin
  envolverlo en un struct solo para cumplir la interfaz.
- `internal/infrastructure/delivery/webapp/response`: construye respuestas JSON
  consistentes, de éxito y de error. `WriteError` es el único lugar que
  traduce un `*domain.Error` a una respuesta HTTP: usa su `Code` y `Message`
  tal cual, loguea su `Cause` (si existe) sin exponerlo, y mapea su `Kind`
  (una clasificación de negocio, no un status HTTP) a un status con una
  función chica y cerrada. Un error que no sea `*domain.Error` se loguea y se
  devuelve como 500 genérico, sin exponer el detalle interno. Los handlers no
  arman ese mapeo por su cuenta.
- `internal/infrastructure/delivery/webapp/route`: registra los endpoints HTTP
  sobre un `*http.ServeMux` a partir de los handlers.
- `internal/infrastructure/delivery/webapp/server`: construye el `*http.Server`
  y el middleware CORS.

### Dirección de dependencias

```mermaid
flowchart LR
    main["cmd/server"] --> app["internal/app"]
    app --> environments["infrastructure/environments"]
    app --> deps["infrastructure/delivery/webapp/dependencies"]
    app --> route["infrastructure/delivery/webapp/route"]
    app --> server["infrastructure/delivery/webapp/server"]
    deps --> repo["infrastructure/repository/postgres"]
    deps --> supabase["infrastructure/auth/supabase"]
    route --> handler["infrastructure/delivery/webapp/handler"]
    route --> middleware["infrastructure/delivery/webapp/middleware"]
    middleware --> supabase
    handler --> response["infrastructure/delivery/webapp/response"]
    handler --> usecaseSystem["business/usecase/system"]
    handler --> usecaseUsers["business/usecase/users"]
    repo -.implementa.-> gateway["business/gateway"]
    supabase -.implementa.-> gateway
    usecaseSystem --> gateway
    usecaseUsers --> gateway
    usecaseSystem --> domain["business/domain"]
    usecaseUsers --> domain
    response --> domain
    gateway --> domain
```

El dominio y los casos de uso no importan sus adaptadores; son los adaptadores
los que importan e implementan los contratos del negocio. Por lo tanto:

- un handler no ejecuta SQL ni contiene reglas del negocio;
- un repositorio no toma decisiones del negocio;
- un caso de uso no conoce `http.Request`, JSON ni tipos concretos de `pgx`;
- las interfaces se definen cerca de quien las consume y se mantienen pequeñas;
- no se crea una interfaz para cada tipo, solamente para un límite real;
- las operaciones de I/O reciben `context.Context` como primer parámetro.

### Convenciones HTTP

- Los endpoints funcionales se publican bajo `/api/v1`.
- Los health checks no se versionan, por ejemplo `/healthz` y `/readyz`.
- Los handlers validan entrada, llaman al caso de uso y mapean la salida.
- Los errores deben tener un formato consistente y no exponer detalles internos:

```json
{
  "code": "lot_not_found",
  "message": "El lote solicitado no existe"
}
```

### Persistencia y pruebas

- Se usa `pgx/v5/pgxpool` con SQL explícito; no se incorpora un ORM sin una
  necesidad concreta.
- Las transacciones se controlan desde el caso de uso cuando una operación de
  negocio requiera atomicidad.
- Los casos de uso se prueban con los fakes de `gateway/gatewayfake`.
- Los handlers se prueban con `httptest`.
- Los repositorios PostgreSQL se prueban como integración contra una base real.

## Frontend

### Estructura objetivo

```text
apps/frontend/src/
├── app/
│   ├── router.tsx
│   ├── AppLayout.tsx           # Sidebar + header + área de contenido
│   ├── Sidebar.tsx             # Navegación lateral con íconos por sección
│   ├── UserMenu.tsx            # Menú de cuenta en el header, conectado a Supabase
│   └── providers.tsx           # Cuando existan providers globales
├── features/
│   ├── auth/
│   │   ├── components/
│   │   │   ├── AppAuthProvider.tsx
│   │   │   ├── AuthStatus.tsx
│   │   │   └── RequireAuth.tsx # Guarda las rutas protegidas del router
│   │   ├── config/
│   │   │   └── supabase-client.ts
│   │   ├── hooks/
│   │   │   └── use-auth.ts     # AuthContext con sesión, login y logout
│   │   ├── lib/
│   │   │   ├── describeAuthError.ts  # Traduce el error de Supabase al usuario
│   │   │   └── resolveDisplayName.ts
│   │   └── pages/
│   │       └── LoginPage.tsx   # Formulario de email y contraseña, en /login
│   ├── system-status/
│   │   ├── api/
│   │   │   └── get-system-info.ts
│   │   ├── components/
│   │   │   └── DatabaseStatus.tsx
│   │   ├── hooks/
│   │   │   └── use-system-info.ts
│   │   ├── pages/
│   │   │   └── MonitorPage.tsx # Diagnóstico del entorno, en /monitor
│   │   └── types.ts
│   └── lots/                   # Ejemplo de funcionalidad futura
│       ├── api/
│       ├── components/
│       ├── pages/
│       ├── hooks/
│       └── types.ts
├── shared/
│   ├── api/
│   │   └── client.ts
│   ├── config/
│   │   └── env.ts
│   ├── ui/
│   └── lib/
├── index.css
└── main.tsx
```

También se crea cada directorio solamente cuando tenga contenido real.

### Dirección de dependencias

```text
app → features → shared
```

- `app` configura y compone la aplicación, las rutas, layouts y providers.
- `features` contiene la UI, acceso a datos y comportamiento de cada
  funcionalidad.
- `shared/api` contiene el cliente HTTP y el tratamiento común de errores.
- `shared/config` centraliza la lectura de variables de entorno.
- `shared/ui` contiene componentes visuales sin reglas de una funcionalidad.
- `shared/lib` contiene funciones reutilizables con un propósito específico; no
  debe convertirse en un directorio genérico de helpers.
- Una feature no importa archivos internos de otra. La composición entre
  funcionalidades ocurre en `app` o en una página.
- Se prefieren imports directos y no se crean archivos `index.ts` globales que
  reexporten gran parte de la aplicación.

### Componentes y estado

- Los componentes y hooks deben ser puros; los efectos sincronizan únicamente
  con sistemas externos.
- El estado visual permanece cerca del componente que lo utiliza.
- Los datos del servidor se mantienen separados del estado visual.
- Context se reserva para información transversal estable, como autenticación o
  tema, y no para cada respuesta de la API.
- Se mantiene `fetch` mientras las necesidades sean simples. Una librería como
  TanStack Query se evaluará cuando haya caché, invalidación, mutaciones o datos
  compartidos entre varias pantallas.
- No se agrega una store global hasta que exista un flujo transversal que la
  justifique.
- Un hook se extrae cuando encapsula comportamiento reutilizable o facilita una
  prueba, no solamente para reducir el tamaño de un archivo.
- Se favorece la composición de componentes y variantes explícitas frente a
  componentes configurados con numerosos booleanos.
- Los tests se colocan junto al código probado con nombres como
  `DatabaseStatus.test.tsx`.

### Diseño responsive

- La app tiene que poder usarse desde el celular, así que los componentes
  nuevos se diseñan mobile-first.
- Con Tailwind esto significa escribir primero las clases sin prefijo
  (aplican a cualquier tamaño) pensando en la pantalla más chica, y agregar
  `sm:`/`md:`/`lg:` solamente para adaptar a pantallas más grandes — nunca al
  revés.
- Antes de dar por terminado un componente nuevo, probarlo al menos en un
  viewport angosto (~375px de ancho) además del tamaño de escritorio.

### Seguridad del frontend

`supabase-js` guarda la sesión (access y refresh token) en `localStorage`.
Sin ninguna mitigación, un XSS en el frontend puede leer esos tokens
directamente vía JS y obtener una sesión completa. Como capa de defensa en
profundidad, un `<meta http-equiv="Content-Security-Policy">` inyectado en
build time restringe de dónde puede cargar scripts/estilos/conexiones la
app.

La política la arma `apps/frontend/vite-plugins/content-security-policy.ts`
(función pura `buildContentSecurityPolicy`, con test unitario al lado) y la
inyecta un plugin de Vite vía el hook `transformIndexHtml` — no vive escrita
a mano en `index.html`, para que no se pueda desincronizar del resto de la
configuración:

```text
default-src 'self';
script-src 'self';
style-src 'self' ['unsafe-inline' solo en dev];
img-src 'self' data:;
font-src 'self';
connect-src 'self' <VITE_SUPABASE_URL exacta> <VITE_API_URL exacta>;
base-uri 'self';
form-action 'self';
object-src 'none';
```

- Va por `<meta>` porque hoy no existe ningún servidor de producción para el
  frontend (no hay `Dockerfile` de prod, ni nginx, ni `vite preview` en uso;
  solo `vite dev` en desarrollo) — no hay dónde emitir un header HTTP
  todavía.
- `connect-src` usa los orígenes **exactos** de `VITE_SUPABASE_URL` y
  `VITE_API_URL` (los mismos defaults que `shared/config/env.ts`, vía
  `shared/config/env-defaults.ts` — una sola fuente de verdad). No hay
  wildcard `*.supabase.co`: cualquiera puede crear un proyecto Supabase
  gratis en ese dominio, así que un wildcard habilitaría exfiltrar la
  sesión del `localStorage` a un proyecto ajeno — exactamente lo que esta
  política busca evitar.
- `style-src` relaja a `'unsafe-inline'` únicamente en dev (`command ===
  'serve'`): Vite inyecta el CSS de Tailwind como un `<style>` inline para
  el hot-reload. En el build de producción el CSS sale a un archivo
  externo (`<link rel="stylesheet">`), así que la excepción no aplica y
  queda excluida.
- `script-src` no necesita ninguna excepción en ningún entorno: la app solo
  carga scripts externos (`<script type="module" src="...">`), tanto en dev
  (`@vite/client`) como en el bundle de producción.
- Como el placeholder ya no depende de que Vite reemplace `%VAR%` en HTML
  estático, no hay forma de que quede sin resolver: la función siempre
  recibe un valor (el de la variable de entorno o el default de
  `env-defaults.ts`).
- **Falta `frame-ancestors`** (y `report-uri`/`report-to`/`sandbox`): el
  spec de CSP ignora esas directivas cuando la política llega por `<meta>`
  en vez de header HTTP. Junto con `X-Frame-Options`,
  `X-Content-Type-Options`, `Referrer-Policy` y HSTS, quedan pendientes para
  cuando exista un servidor de producción real que pueda emitir headers —
  no se agrega ese servidor solo para esto, sería infraestructura
  anticipada sin que el hosting esté decidido.

## Reglas comunes

- Usar el mismo vocabulario para una entidad en base de datos, backend,
  endpoints y frontend.
- No compartir directamente modelos de persistencia con respuestas HTTP.
- No crear carpetas globales `controllers`, `services`, `repositories`,
  `models`, `utils`, `helpers` o `common`.
- No introducir CQRS, event buses, microservicios o abstracciones genéricas sin
  un problema concreto que resolver.
- Cada cambio funcional debe incluir las pruebas y documentación afectadas.
- Una dependencia nueva debe justificar qué problema resuelve.

## Flujo de arranque local

`compose.yaml` define cuatro servicios. Tres están activos; `db` sigue
levantándose con `docker compose up` pero ya no respalda a nadie y se retira
en [#128](https://github.com/LoteoApp/LoteosAPP/issues/128) (ver
[development.md](development.md#arrancar-todo-con-docker)):

- `migrate`: aplica las migraciones pendientes contra la base de Supabase y
  termina correctamente.
- `backend`: inicia la API después de las migraciones.
- `frontend`: inicia Vite después de que el backend esté saludable.

Las migraciones son un proceso separado. Nunca se ejecutan como efecto
secundario de iniciar cada réplica del backend.

## Decisiones vigentes

- Monorepo con pnpm como único package manager de JavaScript.
- React, TypeScript, Vite y Tailwind CSS para el frontend.
- shadcn/ui como base de componentes visuales, instalados en `shared/ui`.
  Configuración manual (sin la CLI) porque el registro de shadcn no es
  accesible desde el entorno de desarrollo asistido; se agregan componentes
  copiando su código fuente cuando haga falta.
- Go para el backend.
- PostgreSQL con `pgxpool` para persistencia.
- Goose y archivos SQL versionados para migraciones, también contra la base
  administrada de Supabase ([#126](https://github.com/LoteoApp/LoteosAPP/issues/126)):
  no se usa el sistema de migraciones propio de Supabase, para no tener dos
  fuentes de verdad del esquema. El SQL Editor y el Table Editor del dashboard
  de Supabase no se usan para cambiar el esquema, ni siquiera puntualmente:
  todo cambio entra como migración Goose revisada en PR. Es una convención de
  equipo, no algo forzado por Supabase: el proyecto usa una única cuenta
  compartida entre los desarrolladores, así que el rol **Read Only** de
  Supabase (que sí bloquearía esto a nivel de plataforma) no aplica — depende
  de logins individuales y además solo existe en los planes Team/Enterprise.
- Arquitectura modular por funcionalidad en ambas aplicaciones.
