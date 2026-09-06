import { apiFetch } from '../../../shared/api/client'
import { isLotState, type LoteOption } from '../types'

const LOTES_PATH = '/api/v1/lotes'
const GENERIC_ERROR = 'No se pudieron cargar los lotes, intentá nuevamente.'

type SearchLotesParams = {
  search?: string
  estado?: string
}

function isLoteResponse(value: unknown): value is Record<string, unknown> {
  if (value === null || typeof value !== 'object') {
    return false
  }

  const candidate = value as Record<string, unknown>
  return (
    typeof candidate.id === 'string' &&
    typeof candidate.manzanaId === 'string' &&
    typeof candidate.loteoId === 'string' &&
    typeof candidate.loteoNombre === 'string' &&
    isLotState(candidate.estado)
  )
}

function toNumberOrNull(value: unknown): number | null {
  return typeof value === 'number' ? value : null
}

function toText(value: unknown): string {
  return typeof value === 'string' ? value : ''
}

function toLoteOption(raw: Record<string, unknown>): LoteOption {
  return {
    id: raw.id as string,
    numero: toText(raw.numero),
    manzanaId: raw.manzanaId as string,
    manzanaNumero: toText(raw.manzanaNumero),
    loteoId: raw.loteoId as string,
    loteoNombre: raw.loteoNombre as string,
    estado: raw.estado as LoteOption['estado'],
    precio: toNumberOrNull(raw.precio),
    moneda: toText(raw.moneda),
    superficie: toNumberOrNull(raw.superficie),
  }
}

export async function searchLotes(
  token: string,
  params: SearchLotesParams = {},
  signal?: AbortSignal,
): Promise<LoteOption[]> {
  const query = new URLSearchParams()
  if (params.search) {
    query.set('q', params.search)
  }
  if (params.estado) {
    query.set('estado', params.estado)
  }

  const suffix = query.toString()
  const body = await apiFetch<unknown>(suffix ? `${LOTES_PATH}?${suffix}` : LOTES_PATH, {
    token,
    signal,
  })

  if (body === null || typeof body !== 'object' || !('lotes' in body)) {
    throw new Error(GENERIC_ERROR)
  }

  const { lotes } = body as { lotes?: unknown }
  if (lotes === null || lotes === undefined) {
    return []
  }
  if (!Array.isArray(lotes) || !lotes.every(isLoteResponse)) {
    throw new Error(GENERIC_ERROR)
  }

  return lotes.map(toLoteOption)
}
