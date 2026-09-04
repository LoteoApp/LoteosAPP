import type { DxfPolygon, LoteoCalle, LoteoDetail, LoteoLote, LoteoManzana } from '../types'

function loteoPolygonId(): string {
  return 'loteo'
}

function manzanaPolygonId(id: string): string {
  return `manzana-${id}`
}

function lotePolygonId(id: string): string {
  return `lote-${id}`
}

function callePolygonId(id: string): string {
  return `calle-${id}`
}

function manzanaLabel(manzana: LoteoManzana): string {
  return manzana.numero ? `Manzana ${manzana.numero}` : 'Manzana'
}

function loteLabel(lote: LoteoLote): string {
  return lote.numero ? `Lote ${lote.numero}` : 'Lote'
}

function calleLabel(calle: LoteoCalle): string {
  return calle.nombre ? `Calle ${calle.nombre}` : 'Calle'
}

// The id prefixes keep a lote and a manzana that share a database UUID from
// colliding on their React key once flattened into one list.
export function planFromLoteoDetail(loteo: LoteoDetail): DxfPolygon[] {
  const polygons: DxfPolygon[] = []

  if (loteo.contorno.length > 0) {
    polygons.push({
      id: loteoPolygonId(),
      layer: 'LOTEO',
      handle: null,
      vertices: loteo.contorno,
      entity: { kind: 'loteo' },
    })
  }

  for (const manzana of loteo.manzanas) {
    if (manzana.poligono.length > 0) {
      polygons.push({
        id: manzanaPolygonId(manzana.id),
        layer: 'MANZANA',
        handle: null,
        vertices: manzana.poligono,
        entity: { kind: 'manzana', id: manzana.id },
        caption: manzana.numero || undefined,
      })
    }
  }

  for (const lote of loteo.lotes) {
    if (lote.poligono.length > 0) {
      polygons.push({
        id: lotePolygonId(lote.id),
        layer: 'LOTES',
        handle: null,
        vertices: lote.poligono,
        entity: { kind: 'lote', id: lote.id },
        caption: lote.numero || undefined,
      })
    }
  }

  for (const calle of loteo.calles) {
    if (calle.poligono.length > 0) {
      polygons.push({
        id: callePolygonId(calle.id),
        layer: 'CALLE',
        handle: null,
        vertices: calle.poligono,
        entity: { kind: 'calle', id: calle.id },
        caption: calle.nombre || undefined,
      })
    }
  }

  return polygons
}

export function planLabelsFromLoteoDetail(loteo: LoteoDetail): Map<string, string> {
  const labels = new Map<string, string>()

  if (loteo.contorno.length > 0) {
    labels.set(loteoPolygonId(), 'Contorno del loteo')
  }

  for (const manzana of loteo.manzanas) {
    if (manzana.poligono.length > 0) {
      labels.set(manzanaPolygonId(manzana.id), manzanaLabel(manzana))
    }
  }

  for (const lote of loteo.lotes) {
    if (lote.poligono.length > 0) {
      labels.set(lotePolygonId(lote.id), loteLabel(lote))
    }
  }

  for (const calle of loteo.calles) {
    if (calle.poligono.length > 0) {
      labels.set(callePolygonId(calle.id), calleLabel(calle))
    }
  }

  return labels
}
