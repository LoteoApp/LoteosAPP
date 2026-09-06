import { ApiError, apiFetch } from '../../../shared/api/client'
import type {
  DxfPoint,
  LoteoCalle,
  LoteoDetail,
  LoteoLote,
  LoteoManzana,
} from '../types'

const LOTEOS_PATH = '/api/v1/loteos'
const GENERIC_ERROR = 'No se pudo cargar el loteo, intentá nuevamente.'

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object'
}

function isPointArray(value: unknown): value is DxfPoint[] {
  return (
    Array.isArray(value) &&
    value.every(
      (point) =>
        isRecord(point) &&
        typeof point.x === 'number' &&
        typeof point.y === 'number',
    )
  )
}

// The backend omits a polygon field (Go `omitempty`) when the entity has no
// DXF ring yet. Absent is fine and maps to []; a present but malformed value
// is a corrupt response and fails the shape check.
function isOptionalPointArray(value: unknown): boolean {
  return value == null || isPointArray(value)
}

function toPolygon(value: unknown): DxfPoint[] {
  return isPointArray(value) ? value.map((point) => ({ x: point.x, y: point.y })) : []
}

export function isManzana(value: unknown): value is Record<string, unknown> {
  return (
    isRecord(value) &&
    typeof value.id === 'string' &&
    typeof value.numero === 'string' &&
    (value.tieneAgua === undefined || typeof value.tieneAgua === 'boolean') &&
    (value.tieneCloaca === undefined || typeof value.tieneCloaca === 'boolean') &&
    (value.tieneLuz === undefined || typeof value.tieneLuz === 'boolean') &&
    (value.tieneGas === undefined || typeof value.tieneGas === 'boolean') &&
    (value.calleIds === undefined || isStringArray(value.calleIds)) &&
    isOptionalPointArray(value.poligono)
  )
}

function isStringArray(value: unknown): value is string[] {
  return Array.isArray(value) && value.every((item) => typeof item === 'string')
}

export function isLote(value: unknown): value is Record<string, unknown> {
  return (
    isRecord(value) &&
    typeof value.id === 'string' &&
    typeof value.manzanaId === 'string' &&
    typeof value.numero === 'string' &&
    (value.precio === null || typeof value.precio === 'number') &&
    typeof value.moneda === 'string' &&
    (value.superficie === null || typeof value.superficie === 'number') &&
    typeof value.caracteristicas === 'string' &&
    isOptionalPointArray(value.poligono)
  )
}

export function isCalle(value: unknown): value is Record<string, unknown> {
  return (
    isRecord(value) &&
    typeof value.id === 'string' &&
    typeof value.nombre === 'string' &&
    typeof value.tipo === 'string' &&
    isOptionalPointArray(value.poligono)
  )
}

function isLoteoDetailResponse(value: unknown): value is Record<string, unknown> {
  if (!isRecord(value)) {
    return false
  }
  return (
    typeof value.id === 'string' &&
    typeof value.nombre === 'string' &&
    typeof value.ubicacion === 'string' &&
    typeof value.descripcion === 'string' &&
    typeof value.fechaCreacion === 'string' &&
    isOptionalPointArray(value.contorno) &&
    Array.isArray(value.manzanas) &&
    value.manzanas.every(isManzana) &&
    Array.isArray(value.lotes) &&
    value.lotes.every(isLote) &&
    Array.isArray(value.calles) &&
    value.calles.every(isCalle)
  )
}

export function toManzana(raw: Record<string, unknown>): LoteoManzana {
  return {
    id: raw.id as string,
    numero: raw.numero as string,
    tieneAgua: raw.tieneAgua === true,
    tieneCloaca: raw.tieneCloaca === true,
    tieneLuz: raw.tieneLuz === true,
    tieneGas: raw.tieneGas === true,
    calleIds: isStringArray(raw.calleIds) ? raw.calleIds : [],
    poligono: toPolygon(raw.poligono),
  }
}

export function toLote(raw: Record<string, unknown>): LoteoLote {
  return {
    id: raw.id as string,
    manzanaId: raw.manzanaId as string,
    numero: raw.numero as string,
    precio: (raw.precio as number | null) ?? null,
    moneda: raw.moneda as string,
    superficie: (raw.superficie as number | null) ?? null,
    caracteristicas: raw.caracteristicas as string,
    poligono: toPolygon(raw.poligono),
  }
}

export function toCalle(raw: Record<string, unknown>): LoteoCalle {
  return {
    id: raw.id as string,
    nombre: raw.nombre as string,
    tipo: raw.tipo as string,
    poligono: toPolygon(raw.poligono),
  }
}

function toLoteoDetail(raw: Record<string, unknown>): LoteoDetail {
  return {
    id: raw.id as string,
    nombre: raw.nombre as string,
    ubicacion: raw.ubicacion as string,
    descripcion: raw.descripcion as string,
    contorno: toPolygon(raw.contorno),
    manzanas: (raw.manzanas as Record<string, unknown>[]).map(toManzana),
    lotes: (raw.lotes as Record<string, unknown>[]).map(toLote),
    calles: (raw.calles as Record<string, unknown>[]).map(toCalle),
    fechaCreacion: raw.fechaCreacion as string,
  }
}

type GetLoteoOptions = {
  signal?: AbortSignal
}

export async function getLoteo(
  loteoId: string,
  token: string,
  { signal }: GetLoteoOptions = {},
): Promise<LoteoDetail> {
  let body: unknown
  try {
    body = await apiFetch<unknown>(
      `${LOTEOS_PATH}/${encodeURIComponent(loteoId)}`,
      { token, signal },
    )
  } catch (error) {
    // ApiError carries a user-facing message and the 404 code the hook maps;
    // an AbortError must keep propagating. Anything else collapses to GENERIC_ERROR.
    if (
      error instanceof ApiError ||
      (error instanceof DOMException && error.name === 'AbortError')
    ) {
      throw error
    }
    throw new Error(GENERIC_ERROR, { cause: error })
  }

  if (!isLoteoDetailResponse(body)) {
    throw new Error(GENERIC_ERROR)
  }

  return toLoteoDetail(body)
}
