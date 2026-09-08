import type { LoteoFieldValues } from '../components/LoteoFields'
import type { DxfPoint, DxfPolygon } from '../types'
import { centroid, distance, pointInPolygon } from './polygonGeometry'

export type EntityPayload = { handle: string; vertices: DxfPoint[] }
export type ManzanaPayload = EntityPayload & { ref: string }
export type LotePayload = EntityPayload & { manzanaRef: string }

export type PlanPayload = {
  loteo: EntityPayload
  manzanas: ManzanaPayload[]
  lotes: LotePayload[]
  calles: EntityPayload[]
}

export type CreateLoteoPayload = {
  nombre: string
  ubicacion: string
  descripcion: string
  plano: PlanPayload | null
}

// BuildLoteoPayloadError is a plan the backend would reject outright (no LOTEO
// ring, lotes with no manzana to attach to). It's surfaced to the user before
// any request goes out.
export class BuildLoteoPayloadError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'BuildLoteoPayloadError'
  }
}

function entityPayload(polygon: DxfPolygon): EntityPayload {
  return { handle: polygon.handle ?? '', vertices: polygon.vertices }
}

// The DXF layers don't record which manzana a lote sits in, and the create
// form has no visual relation editor yet, so the mapping is inferred here:
// the manzana whose polygon contains the lote's centroid, or the nearest one
// when the centroid lands outside every manzana. It can be wrong for oddly
// shaped blocks and is meant to be corrected in a later iteration.
function manzanaRefFor(lote: DxfPolygon, manzanas: DxfPolygon[]): string {
  const loteCenter = centroid(lote.vertices)

  const containing = manzanas.find((manzana) => pointInPolygon(loteCenter, manzana.vertices))
  if (containing) {
    return containing.id
  }

  let closest = manzanas[0]
  let closestDistance = distance(loteCenter, centroid(closest.vertices))
  for (const manzana of manzanas.slice(1)) {
    const candidate = distance(loteCenter, centroid(manzana.vertices))
    if (candidate < closestDistance) {
      closest = manzana
      closestDistance = candidate
    }
  }
  return closest.id
}

export function buildCreateLoteoPayload(
  fields: LoteoFieldValues,
  polygons: DxfPolygon[],
): CreateLoteoPayload {
  const base = {
    nombre: fields.name.trim(),
    ubicacion: fields.location.trim(),
    descripcion: fields.description.trim(),
  }

  if (polygons.length === 0) {
    return { ...base, plano: null }
  }

  const loteoPolygons = polygons.filter((polygon) => polygon.layer === 'LOTEO')
  if (loteoPolygons.length === 0) {
    throw new BuildLoteoPayloadError(
      'El plano necesita un polígono en la capa LOTEO para poder guardarse.',
    )
  }
  if (loteoPolygons.length > 1) {
    throw new BuildLoteoPayloadError('El plano debe tener un único polígono en la capa LOTEO.')
  }
  const [loteoPolygon] = loteoPolygons

  const manzanaPolygons = polygons.filter((polygon) => polygon.layer === 'MANZANA')
  const lotePolygons = polygons.filter((polygon) => polygon.layer === 'LOTES')
  const callePolygons = polygons.filter((polygon) => polygon.layer === 'CALLE')

  if (lotePolygons.length > 0 && manzanaPolygons.length === 0) {
    throw new BuildLoteoPayloadError(
      'El plano tiene lotes pero ninguna manzana; agregá la capa MANZANA al DXF.',
    )
  }

  return {
    ...base,
    plano: {
      loteo: entityPayload(loteoPolygon),
      manzanas: manzanaPolygons.map((polygon) => ({ ...entityPayload(polygon), ref: polygon.id })),
      lotes: lotePolygons.map((polygon) => ({
        ...entityPayload(polygon),
        manzanaRef: manzanaRefFor(polygon, manzanaPolygons),
      })),
      calles: callePolygons.map(entityPayload),
    },
  }
}
