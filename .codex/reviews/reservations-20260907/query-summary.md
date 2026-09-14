# Mediciones SQL de reservas

PostgreSQL 17.6 de desarrollo, configuración inyectada por Doppler. Se creó un esquema aislado con migraciones Goose 1–9, 5.000 reservas activas y 5.010 lotes; un loteo, dos usuarios y un cliente sintéticos. Se ejecutó ANALYZE sobre sus tablas. El esquema fue eliminado al terminar. No se modificaron las tablas comerciales de public.

Cada tiempo de servidor proviene de una ejecución EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON). Los INSERT y SELECT con locks se explicaron dentro de transacciones revertidas. La columna de cliente mide los viajes pgx reales del repositorio e incluye la red; no es tiempo HTTP ni un benchmark de carga. Los pocos samples y la distribución uniforme limitan su extrapolación. Los escenarios que fuerzan bloqueos no sirven para estimar percentiles de producción.

| Consulta | Ejecuciones pgx | Mediana cliente, ms | Planificación, ms | Ejecución servidor, ms | Bloques hit/read |
|---|---:|---:|---:|---:|---:|
| Transición del lote: insertar evento | 10 | 166.66 | 0.046 | 1.156 | 27/0 |
| Vencimiento: insertar evento de reserva | 1 | 332.41 | 0.048 | 0.644 | 15/0 |
| Cancelación: insertar evento de reserva | 4 | 249.35 | 0.024 | 0.708 | 15/0 |
| Alta: insertar reserva y ejecutar triggers | 5 | 167.33 | 0.029 | 1.785 | 24/0 |
| Listado paginado y conteo | 5 | 359.50 | 2.165 | 28.253 | 442/0 |
| Cancelación: revalidar alcance | 4 | 248.87 | 0.458 | 0.085 | 9/0 |
| Transición: bloquear y leer estado del lote | 5 | 166.73 | 0.096 | 0.048 | 4/0 |
| Transición: verificar estado aplicado | 5 | 166.00 | 0.080 | 0.034 | 3/0 |
| Worker: descubrir candidatos por vencimiento | 1 | 331.79 | 0.155 | 0.115 | 4/0 |
| Alta: comprobar idempotencia con bloqueo | 8 | 166.37 | 0.141 | 0.053 | 3/0 |
| Alta: buscar reserva activa del lote | 5 | 165.93 | 0.135 | 0.041 | 2/0 |
| Alta: bloquear lote y loteo | 8 | 166.09 | 0.139 | 0.082 | 6/0 |
| Cancelación: bloquear reserva, vendedor y actor | 4 | 249.61 | 0.240 | 0.075 | 8/0 |
| Cancelación: localizar reserva autorizada | 4 | 248.98 | 0.344 | 0.078 | 9/0 |
| Detalle: historial de estados | 7 | 166.23 | 0.221 | 0.070 | 4/0 |
| Catálogo de vendedores elegibles | 4 | 166.39 | 0.861 | 0.160 | 7/0 |
| Detalle: reserva y datos relacionados | 15 | 166.70 | 1.115 | 0.268 | 11/0 |
| Worker: bloquear lote activo | 1 | 331.30 | 0.083 | 0.047 | 4/0 |
| Cancelación: bloquear lote | 4 | 249.13 | 0.083 | 0.051 | 4/0 |
| Expiración compartida: revalidar reserva | 1 | 331.52 | 0.102 | 0.050 | 4/0 |
| Worker: verificar identidad de reserva activa | 1 | 331.22 | 0.111 | 0.040 | 4/0 |
| Worker: localizar lote y loteo | 1 | 332.39 | 0.135 | 0.060 | 7/0 |
| Worker: bloquear reserva y leer vencimiento | 1 | 331.93 | 0.098 | 0.052 | 4/0 |
| Alta: bloquear actor / vendedor (SQL idéntico) | 13 | 165.83 | 0.075 | 0.047 | 2/0 |
| Alta: validar y bloquear cliente activo | 5 | 166.00 | 0.111 | 0.066 | 2/0 |
| Alta/cancelación: verificar agencia y asignación | — | — | 0.187 | 0.058 | 2/0 |
| Reconciliación de idempotencia después de error | — | — | 0.106 | 0.043 | 2/0 |

Total: **27 formas SQL** del flujo de reservas y de las transiciones reutilizadas. La consulta idéntica de actor y vendedor se agrupa; BEGIN/COMMIT y el lock artificial de la reproducción no se cuentan. Los SQL completos y sus planes están en [query-measurements.json](query-measurements.json) y [additional-query-plans.json](additional-query-plans.json).

## Variantes del listado

| Escenario | Planificación, ms | Ejecución, ms |
|---|---:|---:|
| admin | 9.213 | 49.794 |
| agency | 2.620 | 59.944 |
| development | 1.899 | 23.519 |
| search-diagnostic-fixed | 3.386 | 30.477 |
| search-original | — | ERROR: invalid escape string (SQLSTATE 22025) |

La variante search-diagnostic-fixed corrige únicamente el literal ESCAPE en memoria, para obtener un plan de diagnóstico; el código de la rama no fue modificado. Los tiempos del listado completo oscilaron entre 28,253 y 49,794 ms en las muestras sin scope. Con agencia se midieron 59,944 ms. El conteo de ventana y la ordenación procesan miles de filas para devolver 25. No hay evidencia suficiente para justificar agregar índices de texto indiscriminadamente: primero debe repararse la búsqueda y probar distribución y volumen representativos.

## Operaciones y contención

Las tres altas medidas en el repositorio demoraron **3.161,26 / 1.826,54 / 1.829,47 ms**. Los INSERT con sus triggers demoraron entre 0,644 y 1,785 ms en servidor; el costo de la operación proviene principalmente de los viajes secuenciales a la base. El flujo HTTP agrega la resolución de cuenta/actor y no se cronometró de extremo a extremo.

El SELECT inicial usa FOR UPDATE sobre lotes y loteos. Una transacción sobre el lote A impidió adquirir el lock de otro lote B del mismo loteo: la segunda terminó con 55P03 al fijar lock_timeout=300ms. Hace falta mantener la protección de la baja del loteo con un lock compartido compatible entre reservas, bloquear exclusivamente el lote a modificar y ordenar de forma consistente los locks de usuarios.

El índice parcial de vencimientos se utilizó para descubrir candidatos; esa consulta dio 0,115 ms. El problema del worker es también de progreso: con un primer candidato cuyo lote está de baja y batch=1, dos ejecuciones consecutivas informaron candidates=1, processed=0, skipped=1, failures=0, sin avanzar al siguiente. Con el batch por defecto ocurre si los candidatos que ocupan el lote de trabajo quedan permanentemente sin procesar.
