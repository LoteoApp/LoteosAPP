import { ApiError, apiFetch } from './client'

const AGENCIES_PATH = '/api/v1/inmobiliarias'
const GENERIC_ERROR = 'No se pudo completar la operación, intentá nuevamente.'

// The full shape of an agency as the list endpoint returns it. Consumers
// that only need id/razonSocial (e.g. a picker) can narrow this down instead
// of hitting the backend with their own parsing of the same response.
export type AgencyListItem = {
  id: string
  razonSocial: string
  cuit?: string | null
  telefono?: string | null
  email?: string | null
}

function isOptionalString(value: unknown): boolean {
  return value === undefined || value === null || typeof value === 'string'
}

export function isAgencyListItem(value: unknown): value is AgencyListItem {
  if (value === null || typeof value !== 'object') {
    return false
  }

  const candidate = value as Record<string, unknown>
  return (
    typeof candidate.id === 'string' &&
    typeof candidate.razonSocial === 'string' &&
    isOptionalString(candidate.cuit) &&
    isOptionalString(candidate.telefono) &&
    isOptionalString(candidate.email)
  )
}

export async function listAgencies(token: string, signal?: AbortSignal): Promise<AgencyListItem[]> {
  let body: unknown
  try {
    body = await apiFetch<unknown>(AGENCIES_PATH, { token, signal })
  } catch (error) {
    // ApiError already carries a user-facing message, and an aborted request
    // must keep propagating its AbortError. Anything else (e.g. a 200 whose
    // body isn't JSON) collapses to the generic failure.
    if (error instanceof ApiError || (error instanceof DOMException && error.name === 'AbortError')) {
      throw error
    }
    throw new Error(GENERIC_ERROR, { cause: error })
  }

  if (body === null || typeof body !== 'object' || !('inmobiliarias' in body)) {
    throw new Error(GENERIC_ERROR)
  }

  const { inmobiliarias } = body as { inmobiliarias?: unknown }
  if (inmobiliarias === null || inmobiliarias === undefined) {
    return []
  }
  if (!Array.isArray(inmobiliarias) || !inmobiliarias.every(isAgencyListItem)) {
    throw new Error(GENERIC_ERROR)
  }

  return inmobiliarias
}
