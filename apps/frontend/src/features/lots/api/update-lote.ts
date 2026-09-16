import { ApiError, apiFetch } from '../../../shared/api/client'
import type { UpdateLotePayload } from '../lib/loteFormValues'
import type { LoteoLote } from '../types'
import { isLote, toLote } from './get-loteo'

const GENERIC_ERROR = 'No se pudo guardar el lote, intentá nuevamente.'

export type { UpdateLotePayload }

export async function updateLote(
  loteoId: string,
  loteId: string,
  payload: UpdateLotePayload,
  token: string,
): Promise<LoteoLote> {
  let body: unknown
  try {
    body = await apiFetch<unknown>(
      `/api/v1/loteos/${encodeURIComponent(loteoId)}/lotes/${encodeURIComponent(loteId)}`,
      { method: 'PATCH', body: payload, token },
    )
  } catch (error) {
    if (
      error instanceof ApiError ||
      (error instanceof DOMException && error.name === 'AbortError')
    ) {
      throw error
    }
    throw new Error(GENERIC_ERROR, { cause: error })
  }

  if (!isLote(body)) {
    throw new Error(GENERIC_ERROR)
  }

  return toLote(body)
}
