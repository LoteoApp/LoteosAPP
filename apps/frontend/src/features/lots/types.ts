export type DxfLayer = 'LOTEO' | 'MANZANA' | 'LOTES' | 'CALLE'

export const DXF_LAYERS: readonly DxfLayer[] = ['LOTEO', 'MANZANA', 'LOTES', 'CALLE']

export function isDxfLayer(value: string): value is DxfLayer {
  return (DXF_LAYERS as readonly string[]).includes(value)
}

export type DxfPoint = {
  x: number
  y: number
}

export type DxfPolygon = {
  // Synthetic identifier unique per parse result. The DXF `handle` is not
  // reliable for this: entities copied from a block INSERT all share the
  // same source handle.
  id: string
  layer: DxfLayer
  handle: string | null
  vertices: DxfPoint[]
}

export type DxfValidationIssueCode =
  | 'OPEN_GEOMETRY'
  | 'DEGENERATE_POLYGON'
  | 'SELF_INTERSECTING'
  | 'OVERLAPPING'

export type DxfValidationIssue = {
  code: DxfValidationIssueCode
  layer: DxfLayer
  message: string
  handle: string | null
  // null when the entity was discarded before a DxfPolygon could be built
  // (open geometry, too few vertices).
  polygonId: string | null
  // Present only for OVERLAPPING: the other polygon in the pair.
  relatedPolygonId?: string
}

export type DxfParseResult = {
  polygons: DxfPolygon[]
  issues: DxfValidationIssue[]
}

// LoteoSummary is a loteo as GET /api/v1/loteos returns it: identity and how
// much of a plan it carries. No geometry.
export type LoteoSummary = {
  id: string
  nombre: string
  ubicacion: string
  descripcion: string
  cantidadManzanas: number
  cantidadLotes: number
  cantidadCalles: number
  tienePlano: boolean
}

// The types below mirror GET /api/v1/loteos/{loteoId}. The backend leaves
// `contorno` and every `poligono` off the payload when the entity has no DXF
// ring yet (Go `omitempty`), so the api client normalizes those to [].

export type LoteoManzana = {
  id: string
  numero: string
  poligono: DxfPoint[]
}

export type LoteoLote = {
  id: string
  manzanaId: string
  numero: string
  precio: number | null
  moneda: string
  superficie: number | null
  caracteristicas: string
  poligono: DxfPoint[]
}

export type LoteoCalle = {
  id: string
  nombre: string
  tipo: string
  poligono: DxfPoint[]
}

export type LoteoDetail = {
  id: string
  nombre: string
  ubicacion: string
  descripcion: string
  contorno: DxfPoint[]
  manzanas: LoteoManzana[]
  lotes: LoteoLote[]
  calles: LoteoCalle[]
  fechaCreacion: string
}
