# Plan de implementación: convertir una reserva en venta

Estado: implementado. Desvíos y decisiones posteriores del usuario (administrador
y administrativo convierten cualquier reserva, como en el alta de venta; acceso
desde el visor, el listado y el alta de reserva) quedan documentados en
`docs/domain.md` y `docs/architecture.md`.

Issue: [#224 — Convertir una reserva vigente en venta](https://github.com/LoteoApp/LoteosAPP/issues/224).
Épicas relacionadas: #32 y #36.
Fecha de preparación: 2026-10-02, America/Buenos_Aires.
Base revisada: `develop`, commit `8d99fc9777f2599325c8cfb61124dd130a165385`.

## Instrucciones para quien implemente

Implementar la issue #224 siguiendo este plan y las instrucciones vigentes de
`AGENTS.md`, `CLAUDE.md` y `docs/architecture.md`. Antes de editar, actualizar la
información de la issue y comprobar si cambió `develop`. Crear una rama
`codex/issue-224-reservation-to-sale` desde la base actualizada, sin arrastrar
trabajo ajeno. No iniciar subagentes salvo petición expresa del usuario.

**Decisión posterior del usuario: la issue #217 fue descartada porque el pedido
no era correcto. No aplicar su implementación, su PR #218 ni el trabajo local
guardado de esa rama. No usar #217 como dependencia. Su cierre queda para otra
tarea.** En particular, este trabajo conserva los vendedores administrador,
administrativo e inmobiliaria y no incorpora un filtro nuevo por
`perfil_completo`, ni elimina la venta directa ni cambia los selectores actuales.

El checkout se dejó en `develop` actualizada. El trabajo local previo quedó
preservado en un stash con mensaje
`codex: preserve issue-217 work before issue-224 planning`. No restaurarlo como
parte de esta implementación. Este plan no modifica código, esquema ni datos.

Usar `pnpm`, mantener código/comentarios/identificadores en inglés y textos de
interfaz en español. **No ejecutar el build del frontend Vite localmente**,
aunque aparezca en los comandos generales del repositorio. No añadir
dependencias, un scheduler, una store global ni una API genérica de estados.

## Resultado y alcance

Desde el detalle de una reserva vigente, su vendedor responsable (o, por la
decisión posterior de la regla 2, un administrador o administrativo) puede abrir
«Convertir en venta», consultar los datos comerciales fijos y elegir contado,
financiado o entrega + financiación. Al confirmar, la reserva queda
`convertida`, aparece la venta y el lote pasa a `vendido`; al contado también
se completa la venta y el lote queda `finalizado`. Ningún paso libera el lote.

No incluir reasignación de vendedor, cambio de comprador, prórrogas, señas,
emails, reglas de mora, QR ni cambios al calendario de cuotas de #208.
La reserva actual es sin costo: no crear cobros ficticios ni imputar señas.

## Evidencia de la base actual

| Pieza existente | Hallazgo y reutilización |
| --- | --- |
| `business/domain/reservation.go` | Ya declara `convertida` y estados terminales; contiene errores de reserva y duración de 360 horas. |
| `business/domain/lot_state.go` | Ya admite `reservado -> vendido` con origen `venta`. No cambiar la matriz de estados. |
| `business/usecase/sales/create_sale.go` | Resuelve actor, valida modalidad/plan e idempotencia. Mantener el alta ordinaria con su contrato actual. |
| `repository/postgres/sale.go` | `Create` exige lote disponible; incluye persistencia del plan, cuotas, carga de venta y cierre al contado. |
| `repository/postgres/reservation.go` | Cancelación y expiración bloquean lote antes que reserva y releen el reloj después de los locks. |
| Migraciones `00005`, `00008`, `00009`, `00011`, `00012` | Historiales y triggers existentes, FK compuestas por lote, idempotencia de ventas y finalización al contado. |
| `sales/components/SaleForm.tsx` y `PaymentConditions.tsx` | Selección de participantes y condiciones de pago hoy están acopladas en `SaleForm`; separar la parte reutilizable. |
| `sales/types.ts` | `parsePaymentPlan`, `buildSaleReceipt`, `saleDisabledReason` y cálculo de importes ya existen. |
| `app/ReservationDetailsRoute.tsx` y `ReservationDetailsPage.tsx` | Ya hay detalle, cancelación y comprobante de reserva. |
| `app/SaleCreateRoute.tsx`, `SaleDetailsRoute.tsx` y `router.tsx` | Composición entre features y rutas protegidas disponibles. |

Los paths backend de la tabla son relativos a `apps/backend/internal`; los
frontend, a `apps/frontend/src`. Los DTO actuales son `dto/sales/sales.go` y
`dto/reservations/reservations.go`; no asumir que existen archivos por request.

## Reglas que deben conservarse

1. Solo una reserva `activa`, con `now < fecha_vencimiento`, puede convertirse.
   La igualdad con el vencimiento ya está fuera de plazo. Una demora del worker
   no extiende la reserva.
2. El actor debe ser una cuenta activa y habilitada para ventas **y su ID de
   usuario debe coincidir con `reservas.vendedor_id`**. Comparar UUID de usuario,
   no el `sub` de Supabase ni `usuario_alta` de la reserva. ~~Administrador y
   administrativo pueden convertir si son el vendedor, nunca por su rol solo.~~
   **Reemplazado por decisión posterior del usuario:** administrador y
   administrativo convierten cualquier reserva, como en el alta de venta; si no
   son el vendedor, este tiene que seguir elegible. La inmobiliaria sigue
   convirtiendo solo sus propias reservas.
3. Para actor inmobiliaria, mantener alcance de consulta, agencia activa y
   asignación vigente al loteo exigida para una nueva venta. Un colega de la
   misma agencia puede consultar según el alcance actual, pero no convertir.
4. Lote, cliente y vendedor se resuelven de la reserva. El cliente sigue activo;
   lote y loteo siguen activos; número, precio positivo y moneda son válidos.
   Mantener las reglas vigentes para vendedores internos sin agencia.
5. Precio y moneda se copian del lote al confirmar. El importe mostrado antes
   es informativo; el resultado confirmado y el comprobante usan la venta
   persistida. No enviar precio, monto, moneda ni fechas como autoridad.
6. La fecha de venta y el cálculo de cuotas parten del instante de conversión,
   no de la creación de la reserva. Reutilizar `ValidatePaymentPlan` y
   `BuildPaymentSchedule`; no duplicar fórmulas ni adelantar #208.
7. Una reserva solo origina una venta, incluso si esa venta se cancela después.
   No reactivar ni reutilizar una reserva terminal.

## Contrato HTTP propuesto

Agregar `POST /api/v1/reservas/{id}/convertir`, protegido por autenticación y
cuenta activa, con `Idempotency-Key` obligatorio y timeout comercial existente.

El request contiene exclusivamente `modalidadPago` y `planPago`, con el mismo
formato del alta de ventas. Ejemplo financiado:

```json
{
  "modalidadPago": "financiado",
  "planPago": {
    "cantidadCuotas": 12,
    "tasaInteres": 0,
    "periodicidad": "mensual",
    "montoEntrega": 0
  }
}
```

Contado no admite plan. Conservar los defaults y límites de ventas actuales.
La respuesta exitosa es el `domain.Sale` persistido con `reservaId`, con HTTP
201 también en replay, conforme al patrón de `CreateSaleHandler`.

| Situación | Respuesta |
| --- | --- |
| JSON, modalidad, plan o clave inválidos | 400 con error de dominio/decoder compartido |
| Sin autenticación | 401 desde el middleware |
| Rol no habilitado, cuenta inactiva o actor distinto del vendedor de una reserva visible | 403 |
| Reserva inexistente, ID inválido o fuera del alcance de consulta | 404, sin revelar datos ajenos |
| Reserva cancelada/vencida/convertida, lote incompatible, segunda conversión o clave con otro payload | 409 |
| Fallo de persistencia | Mapeo existente de `ErrDatabaseUnavailable.WithCause`, sin detalles SQL |

Usar `decodeJSON[T]`, `handler.HTTPHandler`, `handler.Adapt` y
`response.WriteError`. El handler depende de un único caso de uso y no abre
transacciones ni crea timeouts. Reutilizar `CreateSalePaymentPlanDTO` dentro de
`dto/sales`, agregando allí el request de conversión. Campos comerciales ajenos
al DTO nunca se usan: probar que un payload manipulado no puede reemplazarlos.
El decoder actual ignora campos desconocidos; no cambiar globalmente esa
semántica como parte de esta issue.

Agregar a las lecturas de reserva `ventaId` opcional y `puedeConvertir` booleano;
a las ventas, `reservaId` opcional. Las ventas ordinarias omiten `reservaId`.
Los vínculos no conceden acceso al recurso relacionado: sus GET conservan el
scope actual. Evitar copiar datos completos de la otra operación.

## Persistencia y migración

Crear una migración Goose nueva, con número disponible al implementar (la
última revisada es `00014`), sin modificar migraciones aplicadas:

- `ventas.reserva_id UUID NULL`, para compatibilidad con todas las ventas
  existentes y el alta ordinaria.
- FK compuesta `(reserva_id, lote_id)` hacia `reservas(id, lote_id)`, cuya
  unicidad ya existe desde `00008`. Así el vínculo no puede apuntar a una
  reserva de otro lote. No usar borrado en cascada.
- Índice único sobre `reserva_id` no nulo, sin condicionar por estado de venta
  ni baja. Cubre unicidad histórica y búsqueda inversa sin otro índice igual.
- Proteger `ventas.reserva_id` frente a cambios posteriores mediante un trigger
  específico pequeño o integración con una protección equivalente si aparece
  antes de implementar. No endurecer todos los campos de ventas en esta tarea.
- Conservar RLS y privilegios vigentes; el frontend no consulta Supabase Data
  API. No crear tablas nuevas ni persistir también `reservas.venta_id`.

El backend valida cliente y vendedor contra la reserva bloqueada; la FK
protege la correspondencia del lote. Persistir cambios de estado insertando
historiales, nunca actualizando directamente `estado_actual`.

Probar Up/Down y actualización de una base con registros previos en un PostGIS
descartable. Down elimina el vínculo y pierde ese metadato: documentarlo y no
usarlo como validación sobre datos reales. No backfillear conversiones por
coincidencia de lote/cliente; no hay evidencia histórica suficiente.

## Caso de uso y operación transaccional

Agregar `ConvertReservationToSale` en
`business/usecase/sales/convert_reservation_to_sale.go`: interfaz `Execute` e
implementación en el mismo archivo. Resolver actor y validar datos propios del
request mediante las reglas existentes. No invocar `CreateSale.Execute` ni
`CancelReservation.Execute` para componer operaciones separadas.

Extender `gateway.SaleRepository` con `ConvertReservation` y un comando propio
que lleve reserva, actor resuelto, identidad de autenticación, términos
normalizados y clave/hash. No aceptar los IDs comerciales desde el cliente.
Actualizar `gatewayfake/sale_repository.go`; no redefinir fakes por test.

Implementar el método en el mismo adaptador `postgres.SaleRepository`, en un
archivo concreto `reservation_sale.go` si mejora la legibilidad. La operación
completa usa un único `pgx.Tx`, con esta secuencia:

1. Resolver las referencias inmutables de la reserva con lectura acotada al
   actor. Esa lectura es para localizar el lote, no autoriza ni decide vigencia.
2. Bloquear loteo activo con lock compartido; lote con `FOR UPDATE`; después
   reserva con `FOR UPDATE OF r`. Usar consultas separadas para controlar el
   orden de adquisición. Revalidar referencias y bajas después de cada espera.
3. Bloquear/revalidar actor; comprobar rol real, cuenta activa, scope y que sea
   el vendedor. No otorgar acceso por una consulta sin scope ni por el JWT solo.
4. Buscar por `(usuario_alta, idempotency_key)`. Si ya está confirmada esa
   conversión con el mismo hash y reserva, cargar su venta y devolverla. Esto
   ocurre **antes** de exigir reserva activa/lote reservado o nuevas condiciones
   de elegibilidad comercial.
5. Para una conversión nueva, revalidar reserva activa, que sea la reserva que
   explica el estado reservado del lote, ausencia de venta previa, cliente,
   vendedor y agencia/asignación cuando corresponda. Bloquear sus filas usando
   los helpers existentes; actor y vendedor coinciden, no crear un ciclo de
   locks innecesario. Respetar lote antes que reserva en todos los participantes.
6. Leer el reloj autoritativo después de adquirir los locks que puedan esperar.
   Si `now >= fecha_vencimiento`, devolver `ErrReservationExpired` sin crear
   venta ni liberar el lote: dejar su regularización al circuito existente.
   Usar ese instante para fecha de venta y generación del plan.
7. Insertar la venta con `reserva_id` y los datos comerciales de la reserva;
   insertar plan/cuotas cuando corresponda. Reutilizar el seed de estado activo
   de ventas y no duplicar su evento inicial.
8. Insertar en `reserva_estados` el evento `convertida` con actor y razón clara;
   usar la protección/trigger existente. Registrar el lote `reservado -> vendido`
   con origen `venta`, `reserva_id` y `venta_id` en el mismo evento.
9. Al contado, reutilizar `settleSale` para `activa -> completada` y
   `vendido -> finalizado`, con sus historiales y razón actuales.
10. Cargar la venta confirmable, hacer commit y responder. Ante cualquier error
    antes de commit, rollback completo mediante `rollbackTransaction`.

La lectura del reloj no debe fijarse en el use case antes de esperar locks.
Agregar un reloj inyectable al repositorio de ventas siguiendo el patrón del
repositorio de reservas, conservando compatibilidad de los constructores
actuales. No introducir un framework de tiempo. Los tests de espera deben
controlar el reloj sin data races.

Extraer únicamente el bloque de escritura de venta/plan que realmente
comparten `Create` y `ConvertReservation`, como función privada que recibe el
`tx` ya abierto y datos normalizados. Reutilizar `insertPaymentPlan`,
`loadSale`, `settleSale` y `transitionLotStateWithLockedLot`. Mantener separados
los guardas de alta disponible y conversión reservada; no volver opcional el
chequeo de estado del alta común. Cubrir la extracción con regresión antes y
después. El vínculo con la reserva debe entrar en el INSERT, no en un UPDATE
posterior.

### Concurrencia, idempotencia y fallos

- Conservar el namespace actual de claves de ventas por actor. El hash de
  conversión incluye un discriminador como `reservation-to-sale:v1`, ID de
  reserva y términos efectivos normalizados. No incluir reloj, estado actual
  ni precio mutable: un replay debe recuperar el monto original persistido.
- La clave no se deriva de la clave de creación de reserva. Una clave usada en
  venta ordinaria, otra reserva o términos distintos devuelve conflicto.
- Un replay sigue exigiendo actor activo, responsable y autorización actual de
  consulta; no exige que la reserva continúe activa ni que el cliente continúe
  elegible para una nueva venta. No devolver resultados ajenos por clave sola.
- El lote serializa conversión, cancelación y worker. Si otro proceso convirtió,
  worker/cancelación deben ver `convertida` después del lock y no liberar nada.
  Si cancelación/expiración ganó, no crear la venta. Las claves nuevas sobre una
  reserva convertida son conflicto; la misma clave devuelve el resultado.
- Revisar los locks de las rutas existentes y extender sus tests; modificar
  únicamente lo necesario para una carrera demostrada. No cambiar el worker
  ni agregar una tarea periódica nueva por defecto.
- Mantener reintentos limitados solo para fallos transitorios. Ante commit
  ambiguo o respuesta perdida, reconciliar por actor, clave, hash y reserva;
  validar permisos también en esa reconciliación. Nunca generar otra clave
  automáticamente ni afirmar fracaso definitivo si se confirmó la venta.
- Mantener transacciones cortas y contextos/timeouts existentes. No generar
  comprobantes ni efectuar llamadas HTTP, emails o almacenamiento bajo locks.

## Lecturas y autorización de acciones

Extender `reservationColumns`, sus JOIN, `scanReservation` y
`scanReservationWithTotal` para resolver el vínculo y `puedeConvertir` sin N+1.
Usar el actor de `ReservationScope.ActorAuthProviderID`, el rol y baja reales,
propietario, estado/plazo, cliente activo, lote completo y las condiciones de
agencia pertinentes. No usar el `puedeCancelar` existente: permite actores
distintos del vendedor. No agregar un filtro por perfil completo.

Las lecturas `Get` y `List` deben publicar una capacidad coherente; para el
plazo usar un instante actual por lectura, evitando el tiempo de comienzo de
una transacción larga. El POST revalida todo aunque la capacidad se haya leído
como verdadera. Para reservas terminales o actor no resuelto es falsa.

La búsqueda inversa puede ser un LEFT JOIN sobre `ventas.reserva_id`, cuyo
índice garantiza cardinalidad uno. Mantener listados, filtros, conteos y scopes.
Extender `saleColumns`/`saleRow` para `reservaId`; revisar consumidores de esos
scanners, incluidos Cobranzas y fixtures, antes de cambiar el orden de columnas.

## Frontend y reutilización

Agregar `/reservas/:id/convertir` en `app/router.tsx` con los roles actuales de
reservas/ventas, y `app/ReservationSaleRoute.tsx` como composición. La ruta
obtiene reserva y loteo para datos/precio/moneda/plano actuales; mapea esos
datos a un contrato mínimo de la feature ventas. Esa feature no importa APIs,
hooks, componentes ni tipos internos de reservas o lotes.

Agregar en `features/sales` una página `ReservationSalePage` y la llamada HTTP
de conversión en `api/sales.ts`. La página recibe contexto fijo, estado de carga,
permiso, callback de conversión y render del plano. No necesita catálogo de
clientes, vendedores ni agencias ni diálogo de alta de cliente.

Refactorización concreta del formulario:

- Extraer de `SaleForm` un `SalePaymentForm` para modalidad, campos del plan,
  preview, validación y confirmación. Reutilizar `PaymentConditions`,
  `parsePaymentPlan`, `buildSaleReceipt` y los tipos existentes.
- `SaleForm` conserva sus selectores y venta directa; entrega sus participantes
  seleccionados al bloque de pago. La nueva página entrega participantes fijos
  y los muestra como datos de solo lectura. No cargar catálogos en esa variante.
- Preferir composición y props explícitas a agregar booleans `isConversion`,
  `hideClient`, `hideSeller`, etc. No copiar todo `SaleForm` ni forzar los datos
  fijos mediante efectos de preselección.
- `SaleCreatePage` continúa aceptando únicamente lotes disponibles. La página
  de conversión evalúa la reserva/capacidad y lote reservado con reglas propias;
  no simular que el lote es disponible para reutilizar esa página.

Agregar «Convertir en venta» desde `ReservationDetailsPage` cuando
`puedeConvertir` sea true, y «Ver venta» para el vínculo de una reserva convertida.
El detalle de venta muestra un acceso a su reserva de origen. Actualizar tipos,
validadores de respuesta y pruebas de las APIs de ambas features.

Separar borrador de condiciones de pago de datos del servidor. Derivar importes
y validaciones durante render; ejecutar mutaciones en submit, no en efectos.
Cancelar lecturas con `AbortSignal` y descartar respuestas de otra reserva o
sesión. El cambio de identidad reinicia la clave/borrador; un refresco corriente
no debe borrar lo escrito.

Generar una clave por intento lógico; bloquear doble submit. Un error de red o
timeout conserva clave y payload para reintentar. Si el resultado es incierto,
no permitir modificar términos y enviar el mismo intento con otra clave sin
resolver primero si hubo conversión. Los errores de negocio conservan el
borrador y ofrecen actualizar la reserva.

Tras recibir la venta confirmada, mostrar éxito basado en su estado real
(`completada`/`finalizado` al contado), permitir acceder a `/ventas/{id}` y al
comprobante existente. Refrescar reserva/lote donde sigan montados; al navegar
de regreso, volver a consultar. Un fallo de ese refresco no revierte el éxito
ni invita a registrar otra venta. No introducir una caché global para esto.

Usar las primitives actuales, etiquetas accesibles, errores anunciados, foco
visible y diseño mobile-first. El estado se expresa con texto, no solo color.

## Secuencia de implementación y archivos

1. **Base y regresión:** releer issue, docs y código; ejecutar suites focalizadas
   de ventas/reservas antes de extraer nada y registrar fallos previos.
2. **Dominio/gateway:** extender `domain/{reservation,sale}.go`,
   `gateway/sale_repository.go` y su fake; agregar el caso de uso de conversión
   y tests. Reutilizar errores existentes; agregar específicos solo si hacen
   falta para permiso de propietario/inconsistencia comercial.
3. **Migración/repositorio:** nueva migración, `postgres/reservation_sale.go`,
   extracción estrecha en `sale.go`, lecturas en `reservation.go` y pruebas
   reales de persistencia/constraints/concurrencia.
4. **HTTP:** `handler/convert_reservation_to_sale.go` y test, request en
   `dto/sales/sales.go`, wiring en `dependencies/dependencies.go` y registro en
   `route/route.go`. Los casos de uso y adapters mantienen sus límites.
5. **Frontend:** contrato/API, extracción del bloque de pagos con regresión,
   página de conversión, ruta en `app` y links en detalles. Pruebas junto a cada
   consumidor antes de avanzar a validación completa.
6. **Documentación/verificación:** actualizar docs afectadas, ejecutar comandos
   y revisar diff completo. Preparar PR contra `develop` con la plantilla y
   `Closes #224`, capturas y resultados reales, cuando se autorice publicarlo.

Ubicaciones propuestas, sin crear directorios genéricos:

| Área | Nuevos archivos principales |
| --- | --- |
| Backend use case | `apps/backend/internal/business/usecase/sales/convert_reservation_to_sale.go` y `_test.go` |
| Persistencia | `apps/backend/internal/infrastructure/repository/postgres/reservation_sale.go` y `_test.go` |
| HTTP | `apps/backend/internal/infrastructure/delivery/webapp/handler/convert_reservation_to_sale.go` y `_test.go` |
| Frontend reusable | `apps/frontend/src/features/sales/components/SalePaymentForm.tsx` y `.test.tsx` |
| Frontend página | `apps/frontend/src/features/sales/pages/ReservationSalePage.tsx` y `.test.tsx` |
| Composición | `apps/frontend/src/app/ReservationSaleRoute.tsx` y `.test.tsx` |

## Matriz mínima de pruebas

| Grupo | Casos y observaciones exigidas |
| --- | --- |
| Conversión válida | Tres modalidades; mismos lote/cliente/vendedor; precio y moneda actuales; vínculo bidireccional; reserva convertida; estado final y cuotas correctos. |
| Roles/propietario | Administrador y administrativo admitidos sobre cualquier reserva (~~internos ajenos rechazados~~, reemplazado por la regla 2), con el vendedor todavía elegible; inmobiliaria dueña admitida con agencia/asignación, colega de agencia rechazado, agrimensor/escribano rechazados. Mantener vendedores internos sin agencia. |
| Vigencia | Antes, exactamente en y después del vencimiento; activa cuyo plazo pasó; cancelada/vencida/convertida; reloj después de espera de lock. |
| Validaciones | Actor no aprovisionado/inactivo, cliente inactivo, agencia inactiva o desasignada, lote/loteo dados de baja, número/precio/moneda incompletos, planes inválidos y payload comercial manipulado. |
| Idempotencia | Misma clave/términos antes y después de conversión o vencimiento original; términos distintos; otra reserva; clave ya usada en alta ordinaria; otra clave sobre convertida; cambios posteriores de precio no alteran replay. |
| Constraints/migración | FK lote incorrecto, segunda venta por reserva incluso tras cancelación de la primera, vínculo inmutable, datos previos con NULL, Up/Down en base descartable. |
| Atomicidad | Provocar un fallo después del INSERT de venta y otro después de escribir historial; no quedan venta/plan/cuotas/vínculo ni cambios de reserva/lote. Commit incierto se reconcilia o queda explícitamente incierto. |
| Carreras | Dos conversiones con misma/diferente clave; conversión contra cancelación y contra worker, incluyendo candidato descubierto antes de convertir; bajas concurrentes relevantes; sin liberación ni eventos duplicados. |
| HTTP | JSON/cabecera/ID, contrato 201 y códigos esperados, auth/cuenta activa, scope, cause oculto y GET de vínculos sin ampliar acceso. |
| UI | Acción visible solo cuando corresponde; datos fijos; modalidades/previews; doble clic; errores con borrador; clave estable en reintento; éxito real al contado; links/recibo; respuesta tardía tras cambiar reserva/sesión. |
| Regresión | Alta disponible, venta directa, venta por colega de agencia vigente en alta ordinaria, reserva/cancelación/expiración, recibos y cobranza de la venta convertida. |

Casos de uso con `gatewayfake`, handlers con `httptest`, UI con Vitest/RTL y
`user-event`. En PostgreSQL usar fixtures aisladas y coordinación mediante
transacciones/canales o espera observable, evitando sleeps arbitrarios. La
suite debe cubrir ambos órdenes de cada carrera, no solo lanzar goroutines.

Los fixtures de conversión deben borrar cobros/cargos/cuotas/planes y ventas
antes que reservas; adaptar cleanup a la nueva FK. Las pruebas que deshabilitan
protecciones append-only o inyectan fallos con triggers se ejecutan únicamente
en una base descartable aislada, nunca en Supabase compartido ni producción.
Seguir el patrón de CI con `postgis/postgis:16-3.4` y migraciones aplicadas.
No contar tests de repositorio saltados como validación de concurrencia.

## Verificación y documentación

Durante implementación, correr primero tests relevantes. Al finalizar:

```powershell
pnpm --filter @loteos/frontend typecheck
pnpm --filter @loteos/frontend lint
pnpm test
pnpm test:coverage
```

Desde `apps/backend`, completar `go vet ./...`; `pnpm test` ya ejecuta
`go test ./...`. Ejecutar integración con `DATABASE_URL` de la base descartable
y migraciones aplicadas; mantener coverage real del repositorio, no solo fakes.
Revisar que `test:backend:coverage` incluya los paquetes tocados (sales, domain,
handlers y postgres ya están incluidos; ampliar solo si se agrega otro paquete).
Mantener 80 % de líneas/statements/functions y 75 % de ramas según corresponda,
y 90 % para permisos, idempotencia y transiciones críticas. No bajar umbrales.
El build de Vite queda para CI, sin ejecutarlo localmente.

Actualizar `docs/domain.md` para quitar «conversión todavía no implementada» y
explicar reglas/actores/plazo; `docs/architecture.md` para operación y ruta;
`docs/database.md` para vínculo, restricciones y rollback; `docs/testing.md`
para fixtures descartables y carreras. Revisar `README.md` si describe ventas
solo sobre disponibles. Conservar fuera de este PR las tareas #217 y #208.

Antes de entregar, comprobar que las diez áreas de la matriz tienen evidencia,
que el POST no confía en participantes/fechas del navegador y que ninguna
conversión termina liberando el lote. Informar cualquier check pendiente con su
motivo. La implementación termina al cumplir #224, sin adelantar otras issues.

## Fuentes técnicas consultadas

- Código y migraciones de la base indicada, issue #224 y documentación del repo.
- Skills locales `supabase`, `supabase-postgres-best-practices` y
  `vercel-react-best-practices`: RLS, índices de FK, orden de locks,
  transacciones cortas, composición y estado derivado.
- [PostgreSQL — bloqueos y deadlocks](https://www.postgresql.org/docs/current/explicit-locking.html): mantener orden de locks y revalidar tras esperar.
- [PostgreSQL — funciones de fecha/hora](https://www.postgresql.org/docs/current/functions-datetime.html): no usar el inicio de transacción como reloj posterior a una espera.
- [PostgreSQL — constraints](https://www.postgresql.org/docs/current/ddl-constraints.html): FK compuesta y unicidad del vínculo.

Este documento especifica decisiones y pruebas a ejecutar; no afirma que la
funcionalidad esté implementada ni que sus suites hayan pasado.
