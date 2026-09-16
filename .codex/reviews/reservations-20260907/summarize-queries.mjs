import { readFileSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
const dir = dirname(fileURLToPath(import.meta.url))
const rows = JSON.parse(readFileSync(join(dir, 'query-measurements.json'), 'utf8'))
const extra = JSON.parse(readFileSync(join(dir, 'additional-query-plans.json'), 'utf8'))
const variants = JSON.parse(readFileSync(join(dir, 'list-variants.json'), 'utf8'))
const median = values => { const a=[...values].sort((a,b)=>a-b); const n=a.length; return n%2?a[(n-1)/2]:(a[n/2-1]+a[n/2])/2 }
function label(sql) {
 if(sql.startsWith('INSERT INTO lote_estados')) return 'Transición del lote: insertar evento'
 if(sql.startsWith('INSERT INTO reserva_estados') && sql.includes("'vencida'")) return 'Vencimiento: insertar evento de reserva'
 if(sql.startsWith('INSERT INTO reserva_estados')) return 'Cancelación: insertar evento de reserva'
 if(sql.startsWith('INSERT INTO reservas')) return 'Alta: insertar reserva y ejecutar triggers'
 if(sql.includes('count(*) OVER()')) return 'Listado paginado y conteo'
 if(sql.startsWith('SELECT EXISTS')) return 'Cancelación: revalidar alcance'
 if(sql.includes('FROM lotes WHERE id = $2')) return 'Transición: bloquear y leer estado del lote'
 if(sql==='SELECT estado_actual FROM lotes WHERE id = $1::uuid') return 'Transición: verificar estado aplicado'
 if(sql.includes('ORDER BY fecha_vencimiento')) return 'Worker: descubrir candidatos por vencimiento'
 if(sql.includes('idempotency_payload_hash')) return 'Alta: comprobar idempotencia con bloqueo'
 if(sql.startsWith('SELECT id::text, fecha_vencimiento')) return 'Alta: buscar reserva activa del lote'
 if(sql.startsWith('SELECT lo.estado_actual')) return 'Alta: bloquear lote y loteo'
 if(sql.startsWith('SELECT r.estado_actual')) return 'Cancelación: bloquear reserva, vendedor y actor'
 if(sql.startsWith('SELECT r.lote_id')) return 'Cancelación: localizar reserva autorizada'
 if(sql.startsWith('SELECT re.id')) return 'Detalle: historial de estados'
 if(sql.startsWith('SELECT u.id')) return 'Catálogo de vendedores elegibles'
 if(sql.startsWith('SELECT r.id')) return 'Detalle: reserva y datos relacionados'
 if(sql.startsWith('SELECT estado_actual FROM lotes') && sql.includes('fecha_baja')) return 'Worker: bloquear lote activo'
 if(sql.startsWith('SELECT estado_actual FROM lotes')) return 'Cancelación: bloquear lote'
 if(sql.startsWith('SELECT estado_actual, fecha_vencimiento')) return 'Expiración compartida: revalidar reserva'
 if(sql.startsWith('SELECT id::text FROM reservas')) return 'Worker: verificar identidad de reserva activa'
 if(sql.startsWith('SELECT lote_id::text, (SELECT')) return 'Worker: localizar lote y loteo'
 if(sql.startsWith('SELECT lote_id::text, estado_actual')) return 'Worker: bloquear reserva y leer vencimiento'
 if(sql.startsWith('SELECT rol')) return 'Alta: bloquear actor / vendedor (SQL idéntico)'
 if(sql.startsWith('SELECT true')) return 'Alta: validar y bloquear cliente activo'
 return sql
}
let md='# Mediciones SQL de reservas\n\n'
md+='PostgreSQL 17.6 de desarrollo, configuración inyectada por Doppler. Se creó un esquema aislado con migraciones Goose 1–9, 5.000 reservas activas y 5.010 lotes; un loteo, dos usuarios y un cliente sintéticos. Se ejecutó ANALYZE sobre sus tablas. El esquema fue eliminado al terminar. No se modificaron las tablas comerciales de public.\n\n'
md+='Cada tiempo de servidor proviene de una ejecución EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON). Los INSERT y SELECT con locks se explicaron dentro de transacciones revertidas. La columna de cliente mide los viajes pgx reales del repositorio e incluye la red; no es tiempo HTTP ni un benchmark de carga. Los pocos samples y la distribución uniforme limitan su extrapolación. Los escenarios que fuerzan bloqueos no sirven para estimar percentiles de producción.\n\n'
md+='| Consulta | Ejecuciones pgx | Mediana cliente, ms | Planificación, ms | Ejecución servidor, ms | Bloques hit/read |\n|---|---:|---:|---:|---:|---:|\n'
let count=0
for(const row of rows){
 const sql=row.SQL.replace(/\s+/g,' ').trim()
 if(['begin','commit'].includes(sql)||sql==='SELECT id FROM lotes WHERE id=$1::uuid FOR UPDATE')continue
 count++
 const p=row.Plan?.[0], plan=p?.Plan
 md+=`| ${label(sql)} | ${row.Calls} | ${median(row.Milliseconds).toFixed(2)} | ${p?.['Planning Time']?.toFixed(3)??'—'} | ${p?.['Execution Time']?.toFixed(3)??row.PlanError} | ${plan?.['Shared Hit Blocks']??'—'}/${plan?.['Shared Read Blocks']??'—'} |\n`
}
for(const [key,value] of Object.entries(extra)){const p=value[0];count++;md+=`| ${key==='agencyAssigned'?'Alta/cancelación: verificar agencia y asignación':'Reconciliación de idempotencia después de error'} | — | — | ${p['Planning Time'].toFixed(3)} | ${p['Execution Time'].toFixed(3)} | ${p.Plan['Shared Hit Blocks']}/${p.Plan['Shared Read Blocks']} |\n`}
md+=`\nTotal: **${count} formas SQL** del flujo de reservas y de las transiciones reutilizadas. La consulta idéntica de actor y vendedor se agrupa; BEGIN/COMMIT y el lock artificial de la reproducción no se cuentan. Los SQL completos y sus planes están en [query-measurements.json](query-measurements.json) y [additional-query-plans.json](additional-query-plans.json).\n\n`
md+='## Variantes del listado\n\n| Escenario | Planificación, ms | Ejecución, ms |\n|---|---:|---:|\n'
for(const [key,v] of Object.entries(variants)){md+=typeof v==='string'?`| ${key} | — | ${v} |\n`:`| ${key} | ${v[0]['Planning Time'].toFixed(3)} | ${v[0]['Execution Time'].toFixed(3)} |\n`}
md+='\nLa variante search-diagnostic-fixed corrige únicamente el literal ESCAPE en memoria, para obtener un plan de diagnóstico; el código de la rama no fue modificado. Los tiempos del listado completo oscilaron entre 28,253 y 49,794 ms en las muestras sin scope. Con agencia se midieron 59,944 ms. El conteo de ventana y la ordenación procesan miles de filas para devolver 25. No hay evidencia suficiente para justificar agregar índices de texto indiscriminadamente: primero debe repararse la búsqueda y probar distribución y volumen representativos.\n\n'
md+='## Operaciones y contención\n\nLas tres altas medidas en el repositorio demoraron **3.161,26 / 1.826,54 / 1.829,47 ms**. Los INSERT con sus triggers demoraron entre 0,644 y 1,785 ms en servidor; el costo de la operación proviene principalmente de los viajes secuenciales a la base. El flujo HTTP agrega la resolución de cuenta/actor y no se cronometró de extremo a extremo.\n\n'
md+='El SELECT inicial usa FOR UPDATE sobre lotes y loteos. Una transacción sobre el lote A impidió adquirir el lock de otro lote B del mismo loteo: la segunda terminó con 55P03 al fijar lock_timeout=300ms. Hace falta mantener la protección de la baja del loteo con un lock compartido compatible entre reservas, bloquear exclusivamente el lote a modificar y ordenar de forma consistente los locks de usuarios.\n\n'
md+='El índice parcial de vencimientos se utilizó para descubrir candidatos; esa consulta dio 0,115 ms. El problema del worker es también de progreso: con un primer candidato cuyo lote está de baja y batch=1, dos ejecuciones consecutivas informaron candidates=1, processed=0, skipped=1, failures=0, sin avanzar al siguiente. Con el batch por defecto ocurre si los candidatos que ocupan el lote de trabajo quedan permanentemente sin procesar.\n'
writeFileSync(join(dir,'query-summary.md'),md)
console.log(`Wrote query-summary.md: ${count} SQL forms`)
