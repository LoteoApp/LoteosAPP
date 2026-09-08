import { ApiError, apiFetch } from '../../../shared/api/client'
import type { LoteoCalle } from '../types'
import { isCalle, toCalle } from './get-loteo'

const GENERIC_ERROR = 'No se pudo guardar la calle, intentá nuevamente.'

export type UpdateCallePayload = {
  nombre: string
  tipo: string
}

export async function updateCalle(
  loteoId: string,
  calleId: string,
  payload: UpdateCallePayload,
  token: string,
): Promise<LoteoCalle> {
  let body: unknown
  try {
    body = await apiFetch<unknown>(
      `/api/v1/loteos/${encodeURIComponent(loteoId)}/calles/${encodeURIComponent(calleId)}`,
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

  if (!isCalle(body)) {
    throw new Error(GENERIC_ERROR)
  }

  return toCalle(body)
}
