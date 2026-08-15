import { Helper } from 'dxf'
import type { DxfLayer, DxfPoint, DxfPolygon } from '../types'

const CLOSED_POLYGON_TYPES = new Set(['LWPOLYLINE', 'POLYLINE'])

// Guards against pathologically large files (many entities to denormalise/tessellate).
const MAX_DXF_LENGTH = 20_000_000

// A DXF INSERT can request a rectangular array of a block via group codes 70
// and 71 (column/row count). The dxf library clones the block's entities
// rows*columns times with no upper bound, so an attacker-crafted file of a
// few KB can request millions of clones and hang the tab.
const MAX_INSERT_ARRAY_COUNT = 10_000

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

// toPolylines() closes rings by repeating the first vertex at the end;
// drop that duplicate to keep each vertex represented once.
function dedupeClosingVertex(vertices: DxfPoint[]): DxfPoint[] {
  if (vertices.length < 2) return vertices
  const first = vertices[0]
  const last = vertices[vertices.length - 1]
  if (first.x === last.x && first.y === last.y) return vertices.slice(0, -1)
  return vertices
}

export function parseDxf(fileContent: string): DxfPolygon[] {
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
  let nextId = 0
  for (let i = 0; i < entities.length; i++) {
    const entity = entities[i]
    if (typeof entity.type !== 'string' || !CLOSED_POLYGON_TYPES.has(entity.type)) continue
    if (entity.closed !== true) continue

    const layer = normalizeLayer(entity.layer)
    if (!layer) continue

    const rawVertices = toVertices(polylines[i]?.vertices)
    if (!rawVertices) continue
    const vertices = dedupeClosingVertex(rawVertices)
    if (vertices.length < 3) continue

    polygons.push({
      id: `${layer}-${nextId++}`,
      layer,
      handle: typeof entity.handle === 'string' ? entity.handle : null,
      vertices,
    })
  }

  if (polygons.length === 0) {
    throw new DxfParseError(
      'No se encontraron polígonos cerrados en las capas LOTEO, MANZANA, LOTES o CALLE.',
    )
  }

  return polygons
}
