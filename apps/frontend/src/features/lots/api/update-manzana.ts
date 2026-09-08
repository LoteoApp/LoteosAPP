import { ApiError, apiFetch } from '../../../shared/api/client'
import type { LoteoManzana } from '../types'
import { isManzana, toManzana } from './get-loteo'

const GENERIC_ERROR = 'No se pudo guardar la manzana, intentá nuevamente.'

export type UpdateManzanaPayload = {
  numero: string
  tieneAgua: boolean
  tieneCloaca: boolean
  tieneLuz: boolean
  tieneGas: boolean
  calleIds: string[]
}

export async function updateManzana(
  loteoId: string,
  manzanaId: string,
  payload: UpdateManzanaPayload,
  token: string,
): Promise<LoteoManzana> {
  let body: unknown
  try {
    body = await apiFetch<unknown>(
      `/api/v1/loteos/${encodeURIComponent(loteoId)}/manzanas/${encodeURIComponent(manzanaId)}`,
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

  if (!isManzana(body)) {
    throw new Error(GENERIC_ERROR)
  }

  return toManzana(body)
}
