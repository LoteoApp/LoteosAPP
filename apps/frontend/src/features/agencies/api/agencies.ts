import { ApiError, apiFetch } from '../../../shared/api/client'
import type { Inmobiliaria, InmobiliariaFormValues } from '../types'

const AGENCIES_PATH = '/api/v1/inmobiliarias'
const GENERIC_ERROR = 'No se pudo completar la operación, intentá nuevamente.'

type AgencyResponse = Pick<Inmobiliaria, 'id' | 'razonSocial'> & {
  cuit?: string | null
  telefono?: string | null
  email?: string | null
}

function isOptionalString(value: unknown): boolean {
  return value === undefined || value === null || typeof value === 'string'
}

function isAgencyResponse(value: unknown): value is AgencyResponse {
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

function toAgency(raw: AgencyResponse): Inmobiliaria {
  return {
    id: raw.id,
    razonSocial: raw.razonSocial,
    cuit: raw.cuit ?? '',
    telefono: raw.telefono ?? '',
    email: raw.email ?? '',
  }
}

type RequestOptions = {
  method?: string
  body?: unknown
  signal?: AbortSignal
}

async function request(token: string, path: string, options: RequestOptions = {}): Promise<unknown> {
  try {
    return await apiFetch<unknown>(path, { ...options, token })
  } catch (error) {
    // ApiError already carries a user-facing message, and an aborted request
    // must keep propagating its AbortError. Anything else (e.g. a 200 whose
    // body isn't JSON) collapses to the generic failure.
    if (error instanceof ApiError || (error instanceof DOMException && error.name === 'AbortError')) {
      throw error
    }
    throw new Error(GENERIC_ERROR, { cause: error })
  }
}

function readAgency(body: unknown): Inmobiliaria {
  if (!isAgencyResponse(body)) {
    throw new Error(GENERIC_ERROR)
  }

  return toAgency(body)
}

export async function listAgencies(
  token: string,
  signal?: AbortSignal,
): Promise<Inmobiliaria[]> {
  const body = await request(token, AGENCIES_PATH, { signal })

  if (body === null || typeof body !== 'object' || !('inmobiliarias' in body)) {
    throw new Error(GENERIC_ERROR)
  }

  const { inmobiliarias } = body as { inmobiliarias?: unknown }
  if (inmobiliarias === null || inmobiliarias === undefined) {
    return []
  }
  if (!Array.isArray(inmobiliarias) || !inmobiliarias.every(isAgencyResponse)) {
    throw new Error(GENERIC_ERROR)
  }

  return inmobiliarias.map(toAgency)
}

export async function createAgency(
  token: string,
  values: InmobiliariaFormValues,
): Promise<Inmobiliaria> {
  return readAgency(await request(token, AGENCIES_PATH, { method: 'POST', body: values }))
}

export async function updateAgency(
  token: string,
  id: string,
  values: InmobiliariaFormValues,
): Promise<Inmobiliaria> {
  return readAgency(
    await request(token, `${AGENCIES_PATH}/${id}`, { method: 'PATCH', body: values }),
  )
}

export async function deleteAgency(token: string, id: string): Promise<void> {
  await request(token, `${AGENCIES_PATH}/${id}`, { method: 'DELETE' })
}
