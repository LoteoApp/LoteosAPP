export type DxfLayer = 'LOTEO' | 'MANZANA' | 'LOTES' | 'CALLE'

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
