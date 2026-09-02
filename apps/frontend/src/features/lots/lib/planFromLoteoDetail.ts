import type { DxfPolygon, LoteoDetail } from '../types'

// The id prefixes keep a lote and a manzana that share a database UUID from
// colliding on their React key once flattened into one list.
export function planFromLoteoDetail(loteo: LoteoDetail): DxfPolygon[] {
  const polygons: DxfPolygon[] = []

  if (loteo.contorno.length > 0) {
    polygons.push({
      id: 'loteo',
      layer: 'LOTEO',
      handle: null,
      vertices: loteo.contorno,
    })
  }

  for (const manzana of loteo.manzanas) {
    if (manzana.poligono.length > 0) {
      polygons.push({
        id: `manzana-${manzana.id}`,
        layer: 'MANZANA',
        handle: null,
        vertices: manzana.poligono,
      })
    }
  }

  for (const lote of loteo.lotes) {
    if (lote.poligono.length > 0) {
      polygons.push({
        id: `lote-${lote.id}`,
        layer: 'LOTES',
        handle: null,
        vertices: lote.poligono,
      })
    }
  }

  for (const calle of loteo.calles) {
    if (calle.poligono.length > 0) {
      polygons.push({
        id: `calle-${calle.id}`,
        layer: 'CALLE',
        handle: null,
        vertices: calle.poligono,
      })
    }
  }

  return polygons
}
