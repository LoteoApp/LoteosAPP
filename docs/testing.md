# Pruebas y cobertura

## Herramientas elegidas

### Backend Go

Se usa la librería estándar de Go:

- `testing` para pruebas unitarias y de integración;
- `net/http/httptest` para probar handlers mediante requests y responses reales;
- `go test -coverprofile` y `go tool cover` para medir cobertura.

Es la opción idiomática para Go, no agrega dependencias y permite probar el
contrato observable del código. Los servicios se prueban con fakes pequeños de
las interfaces que consumen. No se deben simular los detalles internos de
`pgx`; los repositorios SQL se prueban como integración contra PostgreSQL real.

Cuando las primeras funcionalidades de negocio incorporen repositorios SQL, se
evaluará `testcontainers-go` para levantar una instancia aislada de PostgreSQL
por suite. No se incorpora todavía porque el único repositorio actual es el
diagnóstico del entorno y Compose ya permite su verificación integrada.

### Frontend React

Se usan:

- Vitest como runner integrado con Vite y TypeScript;
- React Testing Library para renderizar y consultar la UI;
- `@testing-library/jest-dom` para aserciones del DOM;
- `@testing-library/user-event` para interacciones de usuario;
- jsdom como entorno de navegador para pruebas unitarias y de componentes;
- cobertura V8 mediante `@vitest/coverage-v8`.

Las pruebas deben observar lo mismo que una persona: textos, roles, labels,
estados de carga, errores y resultados. No deben depender del estado interno de
React, nombres de funciones privadas ni clases de Tailwind.

## Umbrales

Para código nuevo o modificado dentro de una funcionalidad:

| Métrica | Mínimo |
| --- | ---: |
| Líneas | 80% |
| Sentencias | 80% |
| Funciones | 80% |
| Ramas | 75% |
| Reglas críticas del negocio | 90% recomendado |

Vitest aplica los umbrales por archivo sobre `src/features`. En Go se revisa la
cobertura de los paquetes del núcleo y handlers de cada feature. Al agregar una
feature backend, se debe incorporarla al comando raíz `test:backend:coverage`.

La cobertura no reemplaza la calidad de los casos. Cada funcionalidad debe
probar, cuando corresponda:

- comportamiento exitoso;
- validaciones y límites;
- errores de dependencias;
- permisos y autenticación;
- estados de carga, vacío y error en la UI;
- regresiones de bugs corregidos.

## Comandos

Desde la raíz:

```powershell
# Backend y frontend
pnpm test

# Reportes de cobertura de ambos proyectos
pnpm test:coverage

# Suites individuales
pnpm test:backend
pnpm test:frontend

# Cobertura individual
pnpm test:backend:coverage
pnpm test:frontend:coverage
```

Modo watch del frontend:

```powershell
pnpm --filter @loteos/frontend test:watch
```

El reporte HTML del frontend se genera en
`apps/frontend/coverage/index.html`. El perfil de Go se genera en
`apps/backend/coverage.out`; ambos están ignorados por Git.

## Ubicación y nombres

- Go: archivo `*_test.go` junto al paquete probado.
- React y TypeScript: archivo `*.test.ts` o `*.test.tsx` junto al módulo o
  componente probado.
- Utilidades globales del entorno frontend: `apps/frontend/src/test`.

No se crean directorios globales con tests desconectados de la funcionalidad.
