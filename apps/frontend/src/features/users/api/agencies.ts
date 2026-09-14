import { apiFetch } from '../../../shared/api/client'

const AGENCIES_PATH = '/api/v1/inmobiliarias'
const GENERIC_ERROR = 'No se pudo completar la operación, intentá nuevamente.'

export type AgencyOption = {
  id: string
  razonSocial: string
}

function isAgencyOption(value: unknown): value is AgencyOption {
  if (value === null || typeof value !== 'object') {
    return false
  }

  const candidate = value as Record<string, unknown>
  return typeof candidate.id === 'string' && typeof candidate.razonSocial === 'string'
}

export async function listAgencies(token: string, signal?: AbortSignal): Promise<AgencyOption[]> {
  const body = await apiFetch<unknown>(AGENCIES_PATH, { token, signal })

  if (body === null || typeof body !== 'object' || !('inmobiliarias' in body)) {
    throw new Error(GENERIC_ERROR)
  }

  const { inmobiliarias } = body as { inmobiliarias?: unknown }
  if (inmobiliarias === null || inmobiliarias === undefined) {
    return []
  }
  if (!Array.isArray(inmobiliarias) || !inmobiliarias.every(isAgencyOption)) {
    throw new Error(GENERIC_ERROR)
  }

  return inmobiliarias
}
