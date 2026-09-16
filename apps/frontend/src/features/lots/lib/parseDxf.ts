import { Helper } from 'dxf'
import type { DxfLayer, DxfParseResult, DxfPoint, DxfPolygon, DxfValidationIssue } from '../types'
import { distance, isSimplePolygon, polygonsOverlap } from './polygonGeometry'

const CLOSED_POLYGON_TYPES = new Set(['LWPOLYLINE', 'POLYLINE'])

// Surveyors commonly close a ring by redrawing back to the start point
// instead of ticking the CAD tool's "closed" flag (DXF group 70), so the
// flag alone under-detects closed rings. Treat a gap this small (survey
// precision, ~1cm) between the first and last vertex as closed too.
const CLOSING_GAP_TOLERANCE = 0.01

// Guards against pathologically large files (many entities to denormalise/tessellate).
const MAX_DXF_LENGTH = 20_000_000

// A DXF INSERT can request a rectangular array of a block via group codes 70
// and 71 (column/row count). The dxf library clones the block's entities
// rows*columns times with no upper bound, so an attacker-crafted file of a
// few KB can request millions of clones and hang the tab.
const MAX_INSERT_ARRAY_COUNT = 10_000
const MAX_OVERLAP_ISSUES = 200

// Surveyors name the layer in singular or plural (e.g. "CALLE" vs "CALLES");
// normalize to the domain's canonical form.
const LAYER_ALIASES: Record<string, DxfLayer> = {
  LOTEO: 'LOTEO',
  MANZANA: 'MANZANA',
  MANZANAS: 'MANZANA',
  LOTE: 'LOTES',
  LOTES: 'LOTES',
  CALLE: 'CALLE',
  CALLES: 'CALLE',
}

export class DxfParseError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'DxfParseError'
  }
}

type RawDxfEntity = {
  type?: unknown
  layer?: unknown
  closed?: unknown
  handle?: unknown
}

type RawDxfPolyline = {
  vertices?: unknown
}

function normalizeLayer(value: unknown): DxfLayer | null {
  if (typeof value !== 'string') return null
  return LAYER_ALIASES[value.trim().toUpperCase()] ?? null
}

// DXF group codes are strictly alternating (code line, value line) for the
// whole document, so this scans the file without needing a full parse.
function assertSafeInsertArrays(fileContent: string): void {
  const lines = fileContent.split(/\r\n|\r|\n/)
  let inInsert = false
  let rowCount = 1
  let columnCount = 1

  const flush = () => {
    if (inInsert && rowCount * columnCount > MAX_INSERT_ARRAY_COUNT) {
      throw new DxfParseError(
        'El archivo DXF contiene un bloque INSERT con un array de copias demasiado grande.',
      )
    }
    inInsert = false
    rowCount = 1
    columnCount = 1
  }

  for (let i = 0; i + 1 < lines.length; i += 2) {
    const code = lines[i].trim()
    const value = lines[i + 1]
    if (code === '0') {
      flush()
      if (value.trim() === 'INSERT') inInsert = true
      continue
    }
    if (!inInsert) continue
    if (code === '70') rowCount = Math.max(1, Number.parseInt(value, 10) || 1)
    if (code === '71') columnCount = Math.max(1, Number.parseInt(value, 10) || 1)
  }
  flush()
}

function toVertices(value: unknown): DxfPoint[] | null {
  if (!Array.isArray(value)) return null

  const vertices: DxfPoint[] = []
  for (const point of value) {
    const [rawX, rawY] = point as [unknown, unknown]
    // The dxf library returns `null` (not NaN) for unparseable coordinates;
    // require an actual finite number rather than coercing with Number().
    if (typeof rawX !== 'number' || typeof rawY !== 'number') return null
    if (!Number.isFinite(rawX) || !Number.isFinite(rawY)) return null
    vertices.push({ x: rawX, y: rawY })
  }
  return vertices
}

// toPolylines() closes rings by repeating the first vertex at the end (for
// flag-closed entities) or the source file already redraws back to the
// start point (for gap-closed ones, see CLOSING_GAP_TOLERANCE); either way,
// drop that duplicate to keep each vertex represented once.
function dedupeClosingVertex(vertices: DxfPoint[]): DxfPoint[] {
  if (vertices.length < 2) return vertices
  const first = vertices[0]
  const last = vertices[vertices.length - 1]
  if (distance(first, last) <= CLOSING_GAP_TOLERANCE) return vertices.slice(0, -1)
  return vertices
}

function isRingClosed(entityClosed: boolean, vertices: DxfPoint[]): boolean {
  if (entityClosed) return true
  if (vertices.length < 2) return false
  return distance(vertices[0], vertices[vertices.length - 1]) <= CLOSING_GAP_TOLERANCE
}

// The overlap search is quadratic per layer, so a plan with thousands of lots
// can produce millions of pairs. Stop at the cap: the list is a warning, not
// an inventory, and past this point it only costs time and memory.
function findOverlapIssues(polygons: DxfPolygon[]): DxfValidationIssue[] {
  const issues: DxfValidationIssue[] = []
  const byLayer = new Map<DxfLayer, DxfPolygon[]>()
  for (const polygon of polygons) {
    const group = byLayer.get(polygon.layer)
    if (group) group.push(polygon)
    else byLayer.set(polygon.layer, [polygon])
  }

  for (const [layer, group] of byLayer) {
    for (let i = 0; i < group.length; i++) {
      for (let j = i + 1; j < group.length; j++) {
        if (issues.length >= MAX_OVERLAP_ISSUES) return issues
        if (!polygonsOverlap(group[i].vertices, group[j].vertices)) continue
        issues.push({
          code: 'OVERLAPPING',
          layer,
          message: `Dos polígonos de la capa ${layer} se superponen.`,
          handle: group[i].handle,
          polygonId: group[i].id,
          relatedPolygonId: group[j].id,
        })
      }
    }
  }

  return issues
}

export function parseDxf(fileContent: string): DxfParseResult {
  if (fileContent.length > MAX_DXF_LENGTH) {
    throw new DxfParseError('El archivo DXF supera el tamaño máximo permitido.')
  }
  assertSafeInsertArrays(fileContent)

  let entities: RawDxfEntity[]
  let polylines: RawDxfPolyline[]
  try {
    const helper = new Helper(fileContent)
    entities = helper.denormalised as RawDxfEntity[]
    // toPolylines() applies block-INSERT transforms (translation, scale,
    // rotation) and tessellates bulges into arc points; denormalised alone
    // only has the raw, untransformed vertices. Both come from the same
    // deterministic pass over the parsed document, so they stay index-aligned.
    polylines = helper.toPolylines().polylines as RawDxfPolyline[]
  } catch (error) {
    if (error instanceof DxfParseError) throw error
    throw new DxfParseError('No se pudo interpretar el archivo DXF.')
  }

  const polygons: DxfPolygon[] = []
  const issues: DxfValidationIssue[] = []
  let nextId = 0
  for (let i = 0; i < entities.length; i++) {
    const entity = entities[i]
    if (typeof entity.type !== 'string' || !CLOSED_POLYGON_TYPES.has(entity.type)) continue

    // Layers outside LOTEO/MANZANA/LOTES/CALLE aren't part of the domain
    // (e.g. MEJORA, LM); stay silent about them, only relevant layers are
    // validated.
    const layer = normalizeLayer(entity.layer)
    if (!layer) continue

    const handle = typeof entity.handle === 'string' ? entity.handle : null

    const rawVertices = toVertices(polylines[i]?.vertices)
    if (!rawVertices) continue

    if (!isRingClosed(entity.closed === true, rawVertices)) {
      issues.push({
        code: 'OPEN_GEOMETRY',
        layer,
        message: `La geometría de la capa ${layer} no está cerrada.`,
        handle,
        polygonId: null,
      })
      continue
    }

    const vertices = dedupeClosingVertex(rawVertices)
    if (vertices.length < 3) {
      issues.push({
        code: 'DEGENERATE_POLYGON',
        layer,
        message: `La geometría de la capa ${layer} tiene menos de 3 vértices.`,
        handle,
        polygonId: null,
      })
      continue
    }

    if (!isSimplePolygon(vertices)) {
      issues.push({
        code: 'SELF_INTERSECTING',
        layer,
        message: `El polígono de la capa ${layer} tiene lados que se cruzan entre sí.`,
        handle,
        polygonId: null,
      })
      continue
    }

    polygons.push({ id: `${layer}-${nextId++}`, layer, handle, vertices })
  }

  if (polygons.length === 0) {
    throw new DxfParseError(
      'No se encontraron polígonos cerrados en las capas LOTEO, MANZANA, LOTES o CALLE.',
    )
  }

  issues.push(...findOverlapIssues(polygons))

  return { polygons, issues }
}
