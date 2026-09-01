import { ApiError, apiFetch } from '../../../shared/api/client'
import type { LoteoSummary } from '../types'

const LOTEOS_PATH = '/api/v1/loteos'
const GENERIC_ERROR = 'No se pudo cargar el listado de loteos, intentá nuevamente.'

type LoteoSummaryResponse = Omit<LoteoSummary, 'descripcion'> & {
  descripcion?: string | null
}

function isLoteoSummaryResponse(value: unknown): value is LoteoSummaryResponse {
  if (value === null || typeof value !== 'object') {
    return false
  }

  const candidate = value as Record<string, unknown>
  return (
    typeof candidate.id === 'string' &&
    typeof candidate.nombre === 'string' &&
    typeof candidate.ubicacion === 'string' &&
    (candidate.descripcion === undefined ||
      candidate.descripcion === null ||
      typeof candidate.descripcion === 'string') &&
    typeof candidate.cantidadManzanas === 'number' &&
    typeof candidate.cantidadLotes === 'number' &&
    typeof candidate.cantidadCalles === 'number' &&
    typeof candidate.tienePlano === 'boolean' &&
    typeof candidate.tieneDxf === 'boolean' &&
    typeof candidate.fechaCreacion === 'string'
  )
}

function toLoteoSummary(raw: LoteoSummaryResponse): LoteoSummary {
  return {
    id: raw.id,
    nombre: raw.nombre,
    ubicacion: raw.ubicacion,
    descripcion: raw.descripcion ?? '',
    cantidadManzanas: raw.cantidadManzanas,
    cantidadLotes: raw.cantidadLotes,
    cantidadCalles: raw.cantidadCalles,
    tienePlano: raw.tienePlano,
    tieneDxf: raw.tieneDxf,
    fechaCreacion: raw.fechaCreacion,
  }
}

type ListLoteosOptions = {
  q?: string
  signal?: AbortSignal
}

export async function listLoteos(
  token: string,
  { q, signal }: ListLoteosOptions = {},
): Promise<LoteoSummary[]> {
  const query = q ? `?q=${encodeURIComponent(q)}` : ''

  let body: unknown
  try {
    body = await apiFetch<unknown>(`${LOTEOS_PATH}${query}`, { token, signal })
  } catch (error) {
    // ApiError already carries a user-facing message; an aborted request must
    // keep propagating its AbortError. Anything else (e.g. a 200 with a body
    // that isn't JSON) collapses to the generic listing failure.
    if (error instanceof ApiError || (error instanceof DOMException && error.name === 'AbortError')) {
      throw error
    }
    throw new Error(GENERIC_ERROR, { cause: error })
  }

  if (body === null || typeof body !== 'object' || !('loteos' in body)) {
    throw new Error(GENERIC_ERROR)
  }

  const { loteos } = body as { loteos?: unknown }
  if (loteos === null || loteos === undefined) {
    return []
  }
  if (!Array.isArray(loteos) || !loteos.every(isLoteoSummaryResponse)) {
    throw new Error(GENERIC_ERROR)
  }

  return loteos.map(toLoteoSummary)
}
