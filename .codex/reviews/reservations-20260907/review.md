# Revisión de la incorporación de reservas

Rama: `codex/implement-reservations`. Base local: `develop` / `origin/develop`, commit `e752b07`. La rama no tenía commits adicionales: se revisaron los cambios sin commit y archivos nuevos, incluida la migración 00009. Fecha: 7 de septiembre de 2026.

**La incorporación necesita correcciones antes de integrarse.** Hay fallos reproducidos en búsqueda, autorización concurrente, vencimiento y recuperación del frontend. La cobertura PostgreSQL incumple el umbral obligatorio.

## Hallazgos

### 1. [P1] La búsqueda de reservas falla ante cualquier texto

[reservation.go:256](E:/LoteosAPP/apps/backend/internal/infrastructure/repository/postgres/reservation.go:256)

El SQL está dentro de un literal raw de Go y usa `ESCAPE '\\'`: PostgreSQL recibe dos caracteres de escape. Una búsqueda no vacía devuelve `invalid escape string`, SQLSTATE `22025`, y se transforma en 503. Reproducido con `Review`, `%` y desde la interfaz con `asd`. Usar un único carácter de escape, como los repositorios existentes, y añadir una integración de búsqueda normal y con caracteres especiales. [Regla de PostgreSQL sobre ESCAPE](https://www.postgresql.org/docs/current/functions-matching.html).

### 2. [P1] La baja de una inmobiliaria no queda protegida durante el alta

[reservation.go:824](E:/LoteosAPP/apps/backend/internal/infrastructure/repository/postgres/reservation.go:824)

`agencyAssigned` lee agencia y asignación sin bloquearlas. Después del chequeo, otra transacción puede dar de baja la agencia o su asignación antes de que la reserva confirme. Reproducción: bloquear el cliente, iniciar un alta de inmobiliaria, esperar a que supere la autorización, dar de baja la agencia y liberar el cliente. **El alta confirmó una reserva activa de la agencia ya dada de baja.** Mantener locks compatibles con las lecturas concurrentes, pero que impidan la revocación mientras se confirma la operación; aplicar un orden común también en cancelación. No basta con validar el catálogo del formulario.

### 3. [P1] Se permite cancelar después del vencimiento al esperar un bloqueo

[reservation.go:376](E:/LoteosAPP/apps/backend/internal/infrastructure/repository/postgres/reservation.go:376), [cancel_reservation.go:62](E:/LoteosAPP/apps/backend/internal/business/usecase/reservations/cancel_reservation.go:62)

`CancelledAt` se captura antes de entrar a la transacción y se compara después de adquirir locks, sin actualizar el instante. Bloqueé el lote, inicié la cancelación antes del vencimiento y liberé el bloqueo después: la operación confirmó `cancelada`, cuando la regla documentada exige `vencida`. La creación usa de igual manera `CreatedAt` para regularizar reservas y calcular el inicio del plazo. Obtener el instante autoritativo después de los locks relevantes, manteniendo el reloj reemplazable en pruebas.

### 4. [P2] Una respuesta de cancelación puede sobrescribir otra reserva

[ReservationDetailsPage.tsx:41](E:/LoteosAPP/apps/frontend/src/features/reservations/pages/ReservationDetailsPage.tsx:41)

La lectura GET descarta respuestas obsoletas, pero `handleCancel` no. Reproduje: cancelar A, cerrar el modal durante el envío, navegar a B y finalmente recibir la respuesta de A. La URL continúa en B y el detalle cambia a A. Validar identidad de reserva y sesión antes de aplicar una mutación resuelta, o desmontar por identidad y proteger los resultados pendientes. La prueba de regresión preparada falla sobre el código actual.

### 5. [P2] Se pierde la identidad de una creación incierta al cerrar el diálogo

[ReservationForm.tsx:55](E:/LoteosAPP/apps/frontend/src/features/reservations/components/ReservationForm.tsx:55), [ReserveLotDialog.tsx:62](E:/LoteosAPP/apps/frontend/src/features/reservations/components/ReserveLotDialog.tsx:62)

La clave idempotente vive en el formulario que se desmonta al cerrar el modal. Simulé una respuesta perdida: al cerrar, reabrir y repetir el mismo alta, se envía una clave distinta. Si la primera operación confirmó, el reintento deja de reconciliarla: obtiene un conflicto o podría crear otra reserva si la anterior ya terminó. Conservar la clave y el payload normalizado mientras el resultado sea incierto; ofrecer reconciliar/reintentar ese intento antes de empezar otro. La prueba de regresión falla mostrando dos UUID distintos.

### 6. [P2] El frontend no consume la paginación de reservas

[ReservationsPage.tsx:20](E:/LoteosAPP/apps/frontend/src/features/reservations/pages/ReservationsPage.tsx:20), [LoteoDetailRoute.tsx:25](E:/LoteosAPP/apps/frontend/src/app/LoteoDetailRoute.tsx:25)

El listado siempre solicita la primera página de 25 y no representa `pagina`, `paginas` ni controles de navegación. A partir de la reserva 26, parte del historial queda inaccesible desde el listado. El visor tiene la misma suposición con un límite de 100 activas y busca la reserva del lote únicamente en esa página: puede ocultar la cancelación de un lote reservado válido. Implementar la navegación del listado y una consulta completa o específica del lote seleccionado para el visor.

### 7. [P2] Una página fuera de rango informa un total incorrecto

[reservation.go:271](E:/LoteosAPP/apps/backend/internal/infrastructure/repository/postgres/reservation.go:271)

El total viene de `count(*) OVER()` en cada fila devuelta. Si OFFSET deja la página vacía, nunca se asigna `page.Total`. Con 5.003 reservas, pedir página 9.999 devolvió `total=0, paginas=0`. Obtener el conteo incluso para una página vacía, conservando exactamente los mismos filtros y alcance. Incluir un test de página posterior al final y de última página que se vacía tras una cancelación/filtro.

### 8. [P2] Las altas de diferentes lotes bloquean el mismo loteo

[reservation.go:52](E:/LoteosAPP/apps/backend/internal/infrastructure/repository/postgres/reservation.go:52)

El `FOR UPDATE` sin `OF` alcanza tanto `lotes` como `loteos`. Lo confirmé bloqueando un lote y consultando otro del mismo loteo: la segunda transacción terminó con `55P03` usando `lock_timeout=300ms`. Con altas de aproximadamente 1,8 segundos, esta serialización aumenta esperas y puede agotar el timeout aunque los usuarios operen lotes distintos. Bloquear exclusivamente el lote que cambia y proteger la vigencia del loteo mediante un lock compartido apropiado; conservar el orden global de locks. [Alcance de los locks de SELECT](https://www.postgresql.org/docs/current/sql-select.html).

### 9. [P2] La API permite reservar sin número ni precio

[reservation.go:53](E:/LoteosAPP/apps/backend/internal/infrastructure/repository/postgres/reservation.go:53), [reservation.go:165](E:/LoteosAPP/apps/backend/internal/infrastructure/repository/postgres/reservation.go:165)

La restricción que deshabilita el botón vive en `ReserveLotDialog`, pero la operación de alta solo comprueba actividad y estado del lote. Eliminé número y precio de un lote de prueba y `Create` confirmó una reserva activa. La regla de habilitación definida en el plan debe validarse también dentro de la operación transaccional, utilizando los datos leídos bajo lock y devolviendo un error de dominio. Añadir una integración para ambos datos ausentes.

### 10. [P2] Candidatos irrecuperables pueden impedir el avance del worker

[reservation.go:461](E:/LoteosAPP/apps/backend/internal/infrastructure/repository/postgres/reservation.go:461), [reservation.go:537](E:/LoteosAPP/apps/backend/internal/infrastructure/repository/postgres/reservation.go:537)

La selección empieza siempre por los primeros vencimientos. Si un candidato queda activo pero su lote está dado de baja, `expireOne` lo omite sin retirarlo del conjunto ni registrar un fallo. Con `batch=1`, dos ejecuciones dieron `processed=0, skipped=1, failures=0` y no alcanzaron las reservas siguientes. Con batch=50, cincuenta candidatos de ese tipo ocupan indefinidamente la ventana. Definir cómo regularizar o excluir esos casos, informar la inconsistencia y recorrer candidatos con avance estable. Un lock largo también consume el timeout compartido del lote de trabajo; acotar el tiempo por candidato ayuda a preservar progreso.

### 11. [P2] Reservas cruza los límites de features del frontend

[ReserveLotDialog.tsx:12](E:/LoteosAPP/apps/frontend/src/features/reservations/components/ReserveLotDialog.tsx:12), [ReservationsPage.tsx:5](E:/LoteosAPP/apps/frontend/src/features/reservations/pages/ReservationsPage.tsx:5)

Hay imports de `clients/hooks/use-clients`, `clients/types`, `lots/types` y `auth/hooks/use-auth` desde reservas. `AGENTS.md` y `docs/architecture.md` los prohíben explícitamente. Mover la composición de sesión y catálogo de clientes a `app`, e inyectar los datos y contratos mínimos que necesita reservas. La acción por render prop entre `lots` y `app` sí respeta el límite; completar esa separación en sus consumidores.

### 12. [P2] Las políticas comerciales están implementadas en PostgreSQL

[reservation.go:835](E:/LoteosAPP/apps/backend/internal/infrastructure/repository/postgres/reservation.go:835)

`cancelActorAllowed`, `isReservationRole`, las decisiones de elegibilidad y buena parte de las reglas de vencimiento están en el adaptador. La arquitectura dice expresamente que un repositorio no toma decisiones de negocio. Mantener en el adaptador la transacción, los locks y la lectura consistente de hechos; extraer las decisiones a funciones/tipos de dominio que pueda invocar dentro de esa transacción. Así la revalidación sigue siendo atómica y las reglas pueden probarse con independencia de PostgreSQL.

### 13. [P2] La cobertura no cumple AGENTS.md

[reservation_test.go:18](E:/LoteosAPP/apps/backend/internal/infrastructure/repository/postgres/reservation_test.go:18)

Con las integraciones habilitadas, `pnpm test:coverage` falla: el paquete PostgreSQL tiene **72,0 %**, frente al 80 % requerido. El archivo nuevo `postgres/reservation.go` cubre **238/413 statements: 57,6 %**. `domain/reservation.go` cubre **23/29: 79,3 %**, por debajo del mínimo general y del 90 % exigido para reglas críticas. La única integración de reservas ejercita principalmente el camino feliz; faltan pruebas permanentes de autorización concurrente, límites temporales, filtros SQL, rollback, fallos y reintentos. Incorporar las reproducciones como regresiones sin bajar umbrales. El porcentaje global 86,4 % no compensa estos incumplimientos.

## Composición, estilo y efectos

La división en formulario, filtros, listado, detalle y diálogos es razonable, y `lots` recibe acciones mediante composición desde `app`. No encontré abuso de `useEffect`: los tres efectos nuevos corresponden a cargas externas con `AbortController`; las mutaciones están en handlers y se deriva buena parte del estado visual. El ajuste de estado durante render por identidad no equivale a una cadena de efectos. [Criterios de React para efectos](https://react.dev/learn/you-might-not-need-an-effect).

Persisten desviaciones de la guía visual: `ReservationForm` define su propio `FieldError` y un `SelectField` nativo, y `ReservationFilters` duplica el select, aunque el plan pide reutilizar `shared/ui/Field`, `FieldError`, `Select`/`Combobox`. El formulario de cancelación marca `aria-invalid` pero no asocia el mensaje con `aria-describedby`. Conviene reemplazar estas piezas por los primitives existentes y eliminar el modo de selección de loteo/lote sin consumidor productivo. El modal también repite el encabezado y los datos del lote dentro de una Card completa.

Los colores usan tokens existentes y clases base con variantes `sm:`/`lg:`. Se inspeccionaron el listado y la apertura del modal a 375 px; el listado tenía `scrollWidth=clientWidth=375`. No se hizo una auditoría exhaustiva de accesibilidad ni se confirmó una operación sobre datos comerciales desde la UI.

## Seguridad y tolerancia a errores

Las rutas están detrás de autenticación y control de cuenta activa; el SQL usa parámetros; los handlers mantienen una dependencia por caso de uso y reutilizan `decodeJSON`, `Adapt` y `WriteError`. La migración agrega un índice idempotente por actor y constraints, y conserva el esquema append-only. Las escrituras de reserva y lote comparten transacción y el rollback usa un contexto de limpieza acotado. Estas garantías no resuelven la carrera de revocación ni el instante temporal obsoleto reproducidos arriba.

Como observación operativa adicional, `reservation_expiry.go:47` registra `err` de un caso de uso que envuelve la causa en `domain.Error`; su método `Error()` solo devuelve el mensaje público. Un fallo al descubrir candidatos puede quedar registrado como “La base de datos no está disponible” sin la causa SQL. Conservar también esa causa en el log estructurado del worker, como ya hace la respuesta HTTP común.

## Rendimiento de consultas

Se midieron **27 formas SQL**, incluidos los statements de escritura y sus triggers, más variantes de scope y búsqueda. [Tabla por consulta, método y límites](query-summary.md).

| Medición | Resultado |
|---|---:|
| Alta completa en repositorio, tres muestras | 1.826,54–3.161,26 ms |
| INSERT y triggers, ejecución en servidor | 0,644–1,785 ms |
| Listado sin scope, 5.000+ reservas | 28,253–49,794 ms SQL |
| Listado de inmobiliaria | 59,944 ms SQL |
| Descubrimiento de vencimientos por índice parcial | 0,115 ms SQL |
| Búsqueda con texto | Error 22025 |

La mayor diferencia en las altas está entre operaciones SQL rápidas y numerosos viajes secuenciales a la base remota. Reducir viajes redundantes y acotar locks aporta más que añadir índices sin evidencia. La muestra tiene distribución sintética uniforme; no acredita capacidad de producción ni percentiles de latencia.

## Verificaciones

| Comprobación | Resultado |
|---|---|
| `pnpm test`, sin DB y luego con DB aislada vía Doppler | Pasa; 100 archivos y 680 tests frontend; integraciones Go ejecutadas en la segunda corrida |
| Migraciones Goose Up/Down | Pasan las pruebas de migración en esquema descartable |
| Frontend typecheck / lint / build | Pasan; cuatro warnings de Fast Refresh y avisos de bundle/imports preexistentes |
| `go vet ./...` | Pasa |
| `pnpm test:coverage` con DB | Falla: PostgreSQL 72,0 % < 80 % |
| `pnpm test:frontend:coverage`, ejecutado por separado después | Pasa; statements 96,68 %, branches 91,69 %, functions 98,88 %, lines 97,43 % |
| Dos regresiones frontend adicionales de esta revisión | Fallan en el código actual: clave incierta y respuesta de cancelación obsoleta |
| Pruebas SQL de búsqueda, reloj, revocación, locks y worker | Reproducciones detalladas arriba |

Se utilizaron los servicios Go/Vite que ya estaban escuchando en 8080/5173 y el CLI de Doppler. No se ejecutaron servicios en Docker. Todas las escrituras de la revisión se limitaron al esquema sintético; se verificó su eliminación. No se implementaron correcciones ni se modificaron los archivos productivos de la rama.

## Evidencia conservada

- [Resultados de reproducciones y limpieza](reproductions.txt).

- [Suite con integraciones](test.log), [cobertura backend](test-coverage.log), [cobertura frontend](frontend-coverage.log).
- [Regresiones frontend](frontend-probes.log), [fuente de las regresiones](frontend-probe.tsx), [configuración de prueba](frontend-probe.config.ts).
- [Planes SQL](query-measurements.json), [planes adicionales](additional-query-plans.json), [variantes del listado](list-variants.json).
- [Programa de medición y fixture](query-audit.go), [reproducciones backend](backend-probes.go).

Los probes fueron temporales y se retiraron de `apps/`; se guardan aquí como evidencia. Para repetirlos, copiar los Go bajo un directorio temporal de `apps/backend/cmd` y el probe/config de Vitest a `apps/frontend`, manteniendo sus rutas relativas. El programa principal crea y elimina su propio esquema; los probes adicionales reciben el nombre de ese esquema mientras exista. Inyectar configuración mediante Doppler. Los archivos de perfil preservados permiten recalcular cobertura sin conectarse a la base.
