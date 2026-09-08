# Plan de implementación: reservas de lotes

> Estado: planificación; funcionalidad pendiente. Decisiones de producto confirmadas el 6 de septiembre de 2026.

## Objetivo y base

Implementar el circuito completo de la épica [#32](https://github.com/LoteoApp/LoteosAPP/issues/32): crear, listar y consultar reservas, cancelarlas con justificación y vencerlas automáticamente liberando el lote. Comprende [#33](https://github.com/LoteoApp/LoteosAPP/issues/33), [#34](https://github.com/LoteoApp/LoteosAPP/issues/34) y [#35](https://github.com/LoteoApp/LoteosAPP/issues/35). La conversión en venta corresponde a la entrega de Ventas.

La base es `develop` con el PR #183 integrado y el [plan de máquina de estados](lote-state-machine.md). Sus apartados de brechas describen la situación anterior a #54: no hay que volver a implementar ese núcleo.

Este documento define trabajo futuro; no acredita que las migraciones o comportamientos propuestos estén desplegados.

## Reglas obligatorias de calidad

[CLAUDE.md](../../CLAUDE.md) declara [AGENTS.md](../../AGENTS.md) como fuente de verdad. Leer ambos y [architecture.md](../architecture.md) antes de cada entrega y verificar su cumplimiento en la revisión.

- Código, identificadores, archivos, tests y commits en inglés; interfaz y mensajes de negocio en español.
- Funciones pequeñas, nombres claros, responsabilidades delimitadas y dependencias explícitas. Sin comentarios que narren el código o decisiones de diseño.
- Reutilizar reglas y componentes existentes antes de crear otros. Extraer comportamiento compartido cuando haya consumidores concretos; evitar tanto copias como frameworks genéricos anticipados.
- Dominio y casos de uso sin HTTP, SQL ni tipos `pgx`. Contratos en `gateway`, fakes compartidos en `gatewayfake` y adaptadores bajo `infrastructure`.
- Un caso de uso con `Execute` por operación; un handler con un solo caso de uso por ruta. DTOs en `dto/reservations`, declarando `package dto`.
- Reutilizar `decodeJSON`, `handler.Adapt` y `response.WriteError`. Errores de negocio `*domain.Error`; conservar la causa técnica para logs sin exponerla.
- Composición frontend en `app`; features sin imports internos cruzados; `shared` solo para elementos realmente transversales. Imports directos y ningún directorio genérico nuevo de helpers o services.
- `pnpm` como único gestor JavaScript. No agregar dependencias ni actualizar versiones como parte incidental de esta funcionalidad.

## Decisiones de producto confirmadas

| Tema | Regla |
| --- | --- |
| Unidad | Un lote individual; sin reservas de manzanas ni loteos completos. |
| Costo | Sin costo ni registro de pago. |
| Plazo | Exactamente 360 horas desde la creación, sin prórroga ni edición manual. |
| Fecha visible | Fecha y hora en `America/Argentina/Buenos_Aires`; persistencia como instante en `TIMESTAMPTZ`. |
| Quién carga | Usuario activo administrador, administrativo o inmobiliaria habilitada para el loteo. |
| Vendedor | Usuario activo con rol administrador, administrativo o inmobiliaria. No se crea un rol vendedor. |
| Selección | Administrador y administrativo eligen vendedor de la lista de usuarios elegibles. Inmobiliaria reserva a su propio nombre. |
| Alcance del vendedor | Si pertenece al rol inmobiliaria, su agencia debe estar activa y asignada al loteo. |
| Auditoría | `usuario_alta` identifica a quien carga; `vendedor_id`, al responsable comercial. |
| Visibilidad | Administrador y administrativo ven todas. Inmobiliaria ve las de su agencia dentro de los loteos asignados. |
| Cancelación | Se conserva lo aprobado en el plan de estados: vendedor responsable, inmobiliaria responsable, administrativo o administrador, con justificación obligatoria y alcance vigente. |

La agencia se obtiene del usuario vendedor según el modelo actual. No introducir una agencia histórica congelada en esta entrega. El historial conserva las referencias aunque usuarios o clientes sean dados de baja; las bajas impiden nuevas operaciones, no borran auditoría.

## Inventario y reutilización

- `migrations/00005_create_entity_model.sql`: tablas `reservas` y `reserva_estados`, índice de una reserva activa por lote, estado inicial e historial append-only.
- `migrations/00008_add_lot_state_machine.sql`: estado vigente del lote, validación de transiciones y FK compuesta que vincula evento y reserva al mismo lote.
- `domain/lot_state.go`: matriz y validación de `LotStateTransition`.
- `repository/postgres/lot_state.go`: `transitionLotState(ctx, tx, command)` reutilizable dentro de una transacción comercial. El método público `Transition` abre su propia transacción: no llamarlo para separar el alta de reserva del cambio del lote.
- Visibilidad de loteos, resolución de usuarios y catálogos existentes: reutilizar contratos y predicados apropiados sin importar un caso de uso desde otro para componer transacciones.
- Frontend: `apiFetch`, `ApiError`, primitives shadcn de `shared/ui`, formateadores y patrones de carga/errores. `use-update-resource` es interno de lotes y está orientado a PATCH: no copiarlo ni importarlo desde reservas. Extraer únicamente piezas generales con una necesidad demostrable y pruebas de los consumidores.
- `/reservas` tiene una pantalla placeholder. El detalle de loteo ya muestra estados; agregar las acciones por composición desde `app`.

## Estados, tiempo y consistencia

| Operación | Reserva | Lote |
| --- | --- | --- |
| Crear | Nueva → activa | disponible → reservado |
| Cancelar antes de vencer | activa → cancelada | reservado → disponible |
| Vencer | activa → vencida | reservado → disponible |
| Convertir, entrega futura | activa → convertida | reservado → vendido |

`cancelada`, `vencida` y `convertida` son terminales para este circuito. No editar directamente `estado_actual`, eliminar reservas ni exponer un PATCH genérico de estados.

Usar un único instante autoritativo del servidor para crear y calcular vencimiento. No confiar en fechas del navegador. Comprobar el plazo después de adquirir locks: una reserva deja de ser operable cuando `now >= fecha_vencimiento`, aunque el worker todavía no haya actualizado el historial. Inyectar un reloj para pruebas de negocio y garantizar una referencia temporal consistente dentro de cada operación.

Las lecturas no deben fingir una liberación persistida: si el procesamiento está atrasado, presentar el vencimiento pendiente como tal. Una nueva reserva sobre ese lote puede regularizar la anterior vencida y crear la nueva dentro de la misma transacción. Nunca renovar la reserva anterior ni reutilizar su identidad.

## Backend y API propuestos

Crear `domain/reservation.go`, el contrato de persistencia de reservas y su fake. Casos de uso en `usecase/reservations`: crear, listar, obtener detalle, cancelar, listar vendedores elegibles y procesar vencimientos. Usar tipos específicos de comando y resultado; las operaciones comerciales del gateway deben expresar atomicidad sin exportar transacciones SQL al negocio.

| Ruta propuesta | Contrato |
| --- | --- |
| `POST /api/v1/loteos/{loteoId}/lotes/{loteId}/reservas` | Cliente y vendedor; clave de idempotencia. Devuelve reserva confirmada y vencimiento. |
| `GET /api/v1/reservas` | Filtros por estado, loteo y búsqueda; listado paginado con orden estable por fecha e ID. |
| `GET /api/v1/reservas/{id}` | Datos de reserva, lote, cliente, vendedor e historial autorizado. |
| `POST /api/v1/reservas/{id}/cancelar` | Justificación obligatoria, actor obtenido de la sesión. |
| `GET /api/v1/loteos/{loteoId}/vendedores` | Catálogo mínimo de usuarios elegibles para ese loteo, sin exponer el ABM completo. |

Reutilizar el catálogo de clientes existente. El endpoint de vendedores es necesario porque el permiso para administrar usuarios no equivale al de seleccionar un vendedor: no ampliar el acceso al ABM para administrativos.

Todas las rutas humanas llevan autenticación y cuenta activa. Validar cliente activo, vendedor elegible, lote y loteo activos, estado y alcance tanto al consultar como al escribir. Inmobiliaria no puede imponer otro vendedor enviando un ID manipulado. Fuera del alcance devolver 404 según el patrón del detalle de loteo, sin revelar datos ajenos.

Validar nuevamente los datos mutables relevantes dentro de la transacción y definir locks coherentes para impedir carreras con bajas y cambios de asignación. El formulario no es una barrera de autorización. Las listas, sus conteos y el historial deben aplicar el mismo scope.

Devolver 201 al crear y 200 al consultar/cancelar; errores esperados 400/401/403/404/409 según el mapeo común. Un conflicto debe permitir refrescar disponibilidad y conservar datos del formulario. No convertir cualquier error inesperado en validación.

## Persistencia, concurrencia e idempotencia

Secuencia de creación:

1. Resolver y validar actor, payload y clave de idempotencia.
2. Abrir transacción corta, proteger el loteo con un lock compartido y bloquear exclusivamente el lote; luego la reserva asociada cuando exista. Mantener ese orden también en cancelación, vencimiento y futuras ventas.
3. Revalidar alcance y elegibilidad. Resolver un reintento ya confirmado antes de crear otra operación.
4. Si corresponde, vencer la reserva anterior y liberar el lote usando el mismo núcleo transaccional de vencimiento.
5. Insertar la reserva; reutilizar el trigger que registra su estado inicial.
6. Invocar la transición de dominio y `transitionLotState` con el mismo `tx`, origen reserva y referencia a la nueva reserva.
7. Confirmar todo junto; responder únicamente con datos confirmados.

Cancelar y vencer insertan el evento de reserva y el evento del lote en la misma transacción. Verificar que la reserva procesada sea la activa que justifica el estado reservado; una referencia histórica al mismo lote no basta para liberar una reserva posterior.

Agregar una migración Goose nueva, con número elegido al implementar, para los cambios realmente necesarios:

- Identificador idempotente de creación, acotado por actor, con unicidad persistida y payload normalizado verificable. Misma clave y mismo payload devuelven la misma reserva; mismo identificador y otro payload producen conflicto. Mantener esta identidad durante la vida del registro para no recrear una reserva ya vencida por un reintento tardío.
- Validación de transiciones de reserva y protección de los campos inmutables del plazo, compatibles con los triggers existentes. Auditar datos previos antes de añadir constraints; no corregir silenciosamente datos incompatibles.
- Índice parcial para reservas activas por vencimiento e ID si los índices existentes no cubren adecuadamente el recorrido del worker.

No modificar migraciones aplicadas ni duplicar tablas/historiales existentes. Conservar RLS y el acceso exclusivamente por backend. Probar Up/Down en una base descartable, documentar pérdida de metadatos si el rollback la implica y no ejecutar Down sobre datos reales como validación.

## Tolerancia a fallos y vencimiento automático

Decisión técnica propuesta: worker Go dentro del ciclo de vida del backend, sin incorporar otro servicio ni scheduler externo. Configuración explícita de habilitación, intervalo, tamaño de lote y timeout; ejecución al iniciar y periódica, inicialmente cada minuto. Un retraso de ejecución no extiende la validez comercial.

- Usar lotes pequeños y transacciones acotadas por reserva. Descubrir candidatos sin bloquear reservas primero; al procesar, bloquear lote y después reserva y volver a comprobar estado/plazo.
- Permitir varias instancias: la segunda revalida tras el lock y omite lo ya procesado. No duplicar eventos. No depender de un mutex en memoria para la exclusión distribuida.
- Registrar origen sistema, referencia de reserva y motivo explícito de vencimiento; actor humano nulo. No suplantar un usuario ni exponer un endpoint público del worker.
- Propagar contextos, hacer rollback explícito y garantizar limpieza aunque el contexto de operación esté cancelado, con un contexto de limpieza acotado cuando corresponda. Manejar fallos de Begin, consultas, Commit y conexión.
- Reintentar fallos transitorios de manera limitada, con espera progresiva y variación aleatoria; reejecutar la transacción completa y conservar idempotencia. No reintentar validaciones, conflictos comerciales o permisos.
- Ante commit ambiguo o respuesta perdida, consultar/repetir la misma operación idempotente. Nunca generar una clave nueva automáticamente para repetir un alta incierta.
- Cancelación repetida de una reserva ya cancelada: devolver el resultado existente, sin otro evento ni sobrescribir la justificación original. Vencida o convertida: conflicto accionable.
- Un candidato inconsistente no debe impedir procesar el resto. Registrar IDs técnicos y causa, sin datos personales innecesarios; informar fallos y atraso mediante logs estructurados y las herramientas operativas disponibles.
- Al reiniciar, recuperar vencimientos pendientes. Apagar worker antes de cerrar pool; ninguna llamada externa mientras se mantiene un lock.

La regularización al reservar y el worker deben invocar la misma operación interna de expiración; no implementar dos versiones de las reglas.

## Frontend: composición y experiencia

Componer sesión, permisos y conexión entre lotes y reservas en `app/ReservationsRoute.tsx` y la ruta de detalle de loteo. La feature `lots` recibe una acción/render prop explícita para reservar; no importa `features/reservations`. Las selecciones provenientes de otra feature se integran mediante callbacks o contratos mínimos en `app`.

Dentro de `features/reservations`, separar API, hooks con responsabilidad concreta, componentes y páginas. Componentes propuestos: `ReservationForm`, `ReservationFilters`, `ReservationsList`, `ReservationDetails`, `ReservationStatusBadge` y `CancelReservationForm`. La página coordina; el formulario controla sus campos; la lista representa datos. Evitar un componente único con múltiples modos controlados por booleanos.

Entrada al flujo: botón Reservar en un lote disponible dentro del visor interactivo del loteo. Al seleccionar un lote, el panel muestra primero sus datos; los roles con permiso deben habilitar explícitamente la edición antes de ver el formulario. La reserva solo se habilita cuando el lote tiene número y precio cargados; si falta alguno, el botón permanece desactivado y explica el motivo en un tooltip. En un lote reservado, los roles habilitados pueden abrir el modal de cancelación con una justificación. Los modales muestran el lote identificado; el de reserva incluye cliente, vendedor y plazo. Después de cada operación se refleja el estado actualizado y la fecha/hora autoritativa. `/reservas` queda destinada a filtros, vacíos, carga, errores, consulta, detalle y cancelación. Conservar las reservas terminales para consulta.

### Estado y uso de efectos

- Mutaciones en handlers de submit/click, nunca en `useEffect` al cambiar un booleano.
- Derivar filtros locales, permisos visuales, selección y validez durante render; no mantener copias sincronizadas de props ni cadenas de efectos.
- Separar datos del servidor del borrador editable. Reiniciar un formulario por identidad mediante `key` o una acción explícita; un refresco no borra silenciosamente lo escrito.
- Reservar efectos para sincronización externa: carga asociada a parámetros/sesión, suscripciones o limpieza. Cancelar lecturas con `AbortSignal` e ignorar respuestas obsoletas tras cambiar entidad, sesión o desmontar.
- Centralizar el refresco tras una mutación en un callback de coordinación: actualizar reserva, listado y disponibilidad afectada sin cascadas de efectos.
- Mantener `fetch` y los patrones existentes inicialmente. Justificar una librería de caché solo si la implementación demuestra necesidad concreta; no añadir una store global para este formulario.
- Estado de operación explícito: reposo, envío, éxito, error y resultado incierto. Bloquear doble submit local, conservando idempotencia en servidor como defensa real.
- Si falla el refresco posterior a una creación confirmada, informar que la reserva se creó y ofrecer recargar; no invitar a crearla de nuevo.

### Estilo y accesibilidad

Usar la configuración existente `base-nova`, Base UI, colores neutrales, variables CSS de `src/index.css`, tipografía Inter e iconos Lucide. Respetar espaciados, radios y patrones de las pantallas actuales.

Reutilizar shadcn `Button`, `Field`, `Input`, `Combobox`, `Select`, `Alert`, `Card`, `Table` y los avisos existentes. Agregar primitives faltantes en `shared/ui` siguiendo el procedimiento documentado; no copiar ejemplos Radix incompatibles con la base instalada ni crear otra biblioteca visual.

Diseño mobile-first con clases base para celular y variantes responsive posteriores. Etiquetas asociadas a campos, errores con `FieldError`/`aria-invalid` y descripciones vinculadas, foco visible y manejo de foco después de errores o cierre de diálogo. Estado expresado con texto además de color. Selección accesible por teclado y pantalla usable sin desbordamiento horizontal obligatorio en móvil.

No duplicar `LotStateBadge` entre features: mantenerlo en lotes cuando solo lo usa esa feature, o extraerlo a un módulo compartido concreto si hay un segundo consumidor real. El estado de reserva tiene semántica propia y su componente permanece en reservas.

## Pruebas y criterios de aceptación

- Dominio: estados terminales, razón requerida, roles y límites exactos del plazo; incluir igualdad con vencimiento y fechas entre meses/años.
- Casos de uso: cada rol permitido/prohibido, actor no aprovisionado, bajas, cliente inválido, vendedor no elegible, agencia ajena y filtros scoped. Fakes reutilizados desde `gatewayfake`.
- HTTP: contrato JSON, autenticación, códigos esperados, payload manipulado, idempotencia y ausencia de filtraciones en listados/detalle/catálogos.
- PostgreSQL real: dos altas simultáneas dejan una reserva activa; rollback entre ambas escrituras; dos reintentos dejan un evento; conflicto de clave con otro payload; cancelación y worker concurrentes; nueva reserva tras vencer; un worker tardío no libera una reserva nueva; recuperación tras interrupción.
- Validar historial append-only, nuevas constraints, plazo inmutable y compatibilidad con el motor existente. No dar por ejecutadas las integraciones si se omitieron por falta de `DATABASE_URL`.
- UI con Vitest, RTL y user-event: ambos puntos de entrada, elección por rol, búsquedas, errores de campos, conflicto, cancelación, doble clic, respuesta perdida, cambio de selección/sesión y fallo de refresco tras éxito. Probar comportamiento observable, sin assertions de clases ni estado interno.
- Probar worker con reloj/control de ejecución determinista, sin sleeps de 15 días: duplicación de instancias, candidatos defectuosos, fallos transitorios, reinicio y apagado.

Umbrales: mínimo 80 % de líneas, statements y funciones, 75 % de ramas; reglas comerciales críticas al menos 90 %. No bajar umbrales ni excluir código para cumplirlos. Extender `test:backend:coverage` con los paquetes incorporados.

La entrega se acepta cuando la operación comercial y ambos historiales son atómicos; el mismo intento no crea duplicados; nadie ve u opera reservas fuera de su scope; el plazo no depende del navegador o del worker; el formulario se reutiliza; y las comprobaciones anteriores tienen evidencia.

## Secuencia de entregas y verificación

1. Contratos, reglas, migración e idempotencia; reutilización transaccional y pruebas de concurrencia.
2. API de alta, lectura y vendedores; composición UI y formulario/listado/detalle de #33.
3. Cancelación, núcleo de expiración, worker y regularización al reservar para #34/#35. Compartir la misma operación de expiración; entregar estas capacidades antes de habilitar reservas en producción.
4. Verificación integrada, accesibilidad móvil, recuperación ante fallos y documentación operativa.

No dar por terminada la épica con solo el alta. En cada PR respetar el template e incluir `Refs` o `Closes` según el alcance real; mantener el tablero alineado con la entrega sin confundir abierto en GitHub con pendiente de merge a develop.

Ejecutar los checks requeridos: `pnpm test`, `pnpm test:coverage`, typecheck/lint/build del frontend, `go test ./...` y `go vet ./...` en backend, y `docker compose config`. Verificar arranque con `docker compose up --build` y migraciones en entorno de prueba apropiado. Revisión manual en celular/escritorio y dos sesiones concurrentes. Registrar explícitamente verificaciones no ejecutables.

Actualizar `docs/domain.md`, `docs/architecture.md`, `docs/database.md`, `docs/testing.md` y `docs/development.md` según las piezas entregadas, incluyendo configuración del worker y recuperación del atraso. Revisar README antes de push. Este plan no implica ejecutar migraciones ni habilitar procesos sobre una base compartida durante su redacción.

## Documentación consultada con Context7

Consulta realizada el 6 de septiembre de 2026, contrastada con las dependencias y archivos locales. No se propone una actualización de paquetes.

- React, biblioteca `/reactjs/react.dev`: [You Might Not Need an Effect](https://react.dev/learn/you-might-not-need-an-effect). Fundamenta estado derivado, handlers de eventos, reinicio por identidad y limpieza de lecturas.
- shadcn/ui, biblioteca `/shadcn-ui/ui`: [Field con Base UI](https://ui.shadcn.com/docs/components/base/field). Fundamenta composición de campos y errores accesibles. Los ejemplos con librerías de formularios no obligan a agregarlas.
- pgx, biblioteca `/jackc/pgx`: [transacciones](https://pkg.go.dev/github.com/jackc/pgx/v5#Tx) y [pgxpool.Begin](https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool#Pool.Begin). Fundamenta control explícito de transacciones y limpieza; comprobar firmas y errores contra la versión v5 instalada, sin copiar detalles de la rama master que puedan corresponder a otra versión.

Fuentes locales adicionales: `CLAUDE.md`, `AGENTS.md`, documentación de dominio/arquitectura/base, `apps/frontend/components.json`, `package.json`, tokens CSS y adaptador de estados actual. Revalidar documentación de APIs adicionales en Context7 si la implementación incorpora capacidades no contempladas aquí.
