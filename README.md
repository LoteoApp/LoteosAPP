# LoteosAPP

Monorepo de LoteosAPP con una aplicación React/Vite, una API Go y PostgreSQL.

## Inicio rápido

Requisitos:

- Docker Desktop con Docker Compose.
- Node.js 20.19+ y pnpm 10+ para trabajar con el frontend fuera de Docker.
- Go 1.26+ para trabajar con el backend fuera de Docker.

Para levantar todo el entorno de desarrollo:

```powershell
docker compose up --build
```

El comando inicia PostgreSQL, aplica las migraciones pendientes, inicia el
backend y finalmente inicia el frontend.

Servicios disponibles:

- Frontend: http://localhost:5173
- Backend: http://localhost:8080
- Liveness: http://localhost:8080/healthz
- Readiness de PostgreSQL: http://localhost:8080/readyz
- Diagnóstico de conexión: http://localhost:8080/api/v1/system
- PostgreSQL: `localhost:5432`

Las credenciales locales por defecto son `loteosapp` / `loteosapp` para la base
`loteosapp`. Se pueden sobrescribir con `POSTGRES_DB`, `POSTGRES_USER`,
`POSTGRES_PASSWORD`, `POSTGRES_PORT`, `BACKEND_PORT` y `FRONTEND_PORT`.

## Comandos frecuentes

```powershell
# Ver estado
docker compose ps

# Ver logs de un servicio
docker compose logs -f backend

# Detener contenedores y conservar los datos de PostgreSQL
docker compose down

# Detener y eliminar también el volumen de PostgreSQL (borra los datos locales)
docker compose down -v

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
migrations/        # Migraciones SQL versionadas de PostgreSQL
compose.yaml       # Entorno completo de desarrollo
```

## Documentación

- [Índice de documentación](docs/README.md)
- [Desarrollo y Docker Compose](docs/development.md)
- [PostgreSQL y migraciones](docs/database.md)
- [Arquitectura y estructura](docs/architecture.md)
- [Pruebas y cobertura](docs/testing.md)
- [Integración continua](docs/ci.md)
- [Reglas para agentes y colaboradores](AGENTS.md)
