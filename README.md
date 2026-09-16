# LoteosAPP

Monorepo de LoteosAPP con una aplicación React/Vite y una API Go. La base de
datos y la autenticación son el proyecto administrado de Supabase. Compose
todavía define un servicio `db` con PostgreSQL, pero ya no lo usa nadie y se
retira en [#128](https://github.com/LoteoApp/LoteosAPP/issues/128).

## Inicio rápido

Requisitos:

- Docker Desktop con Docker Compose.
- [Doppler CLI](https://doppler.com) para los secrets (`SUPABASE_URL`,
  `DATABASE_URL`, etc.), ver [docs/secrets.md](docs/secrets.md).
- Node.js 20.19+ y pnpm 10+ para trabajar con el frontend fuera de Docker.
- Go 1.26+ para trabajar con el backend fuera de Docker.

Para levantar todo el entorno de desarrollo:

```powershell
doppler run -- docker compose up --build
```

El comando aplica las migraciones pendientes contra la base de Supabase,
inicia el backend y finalmente inicia el frontend.

Servicios disponibles:

- Frontend: http://localhost:5173
- Backend: http://localhost:8080

Se pueden sobrescribir `BACKEND_PORT` y `FRONTEND_PORT`.

## Comandos frecuentes

```powershell
# Ver estado
docker compose ps

# Ver logs de un servicio
docker compose logs -f backend

# Detener contenedores
docker compose down

# Verificar el frontend sin Docker
pnpm install
pnpm dev
pnpm build

# Ejecutar todas las pruebas y comprobar cobertura
pnpm test
pnpm test:coverage
```

## Estructura

```text
apps/
├── backend/       # API Go y comandos de migración
└── frontend/      # React + TypeScript + Vite + Tailwind CSS + shadcn/ui
docs/              # Documentación técnica
migrations/        # Migraciones SQL versionadas de PostgreSQL (Supabase)
compose.yaml       # Entorno completo de desarrollo
```

## Documentación

- [Índice de documentación](docs/README.md)
- [Desarrollo y Docker Compose](docs/development.md)
- [PostgreSQL y migraciones](docs/database.md)
- [Secrets con Doppler](docs/secrets.md)
- [Arquitectura y estructura](docs/architecture.md)
- [Pruebas y cobertura](docs/testing.md)
- [Integración continua](docs/ci.md)
- [Reglas para agentes y colaboradores](AGENTS.md)
