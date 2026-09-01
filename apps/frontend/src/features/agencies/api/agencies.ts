import { apiUrl } from '../../../shared/config/env'
import type { Inmobiliaria, InmobiliariaFormValues } from '../types'

const AGENCIES_PATH = '/api/v1/inmobiliarias'
const GENERIC_ERROR = 'No se pudo completar la operación, intentá nuevamente.'

type AgencyResponse = Pick<Inmobiliaria, 'id' | 'razonSocial'> & {
  cuit?: string | null
  telefono?: string | null
  email?: string | null
}

function isAgencyResponse(value: unknown): value is AgencyResponse {
  if (value === null || typeof value !== 'object') {
    return false
  }

  const candidate = value as Record<string, unknown>
  return typeof candidate.id === 'string' && typeof candidate.razonSocial === 'string'
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

async function readErrorMessage(response: Response): Promise<string> {
  try {
    const body: unknown = await response.json()
    if (body && typeof body === 'object' && 'message' in body) {
      const { message } = body as { message?: unknown }
      if (typeof message === 'string' && message !== '') {
        return message
      }
    }
  } catch {
    return GENERIC_ERROR
  }

  return GENERIC_ERROR
}

async function readJSON(response: Response): Promise<unknown> {
  try {
    return await response.json()
  } catch {
    throw new Error(GENERIC_ERROR)
  }
}

async function readAgency(response: Response): Promise<Inmobiliaria> {
  const body = await readJSON(response)
  if (!isAgencyResponse(body)) {
    throw new Error(GENERIC_ERROR)
  }

  return toAgency(body)
}

async function send(token: string, path: string, init: RequestInit = {}): Promise<Response> {
  const response = await fetch(`${apiUrl}${path}`, {
    ...init,
    headers: {
      ...(init.body === undefined ? {} : { 'Content-Type': 'application/json' }),
      Authorization: `Bearer ${token}`,
    },
  })

  if (!response.ok) {
    throw new Error(await readErrorMessage(response))
  }

  return response
}

export async function listAgencies(
  token: string,
  signal?: AbortSignal,
): Promise<Inmobiliaria[]> {
  const response = await send(token, AGENCIES_PATH, { signal })
  const body = await readJSON(response)

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
  const response = await send(token, AGENCIES_PATH, {
    method: 'POST',
    body: JSON.stringify(values),
  })

  return readAgency(response)
}

export async function updateAgency(
  token: string,
  id: string,
  values: InmobiliariaFormValues,
): Promise<Inmobiliaria> {
  const response = await send(token, `${AGENCIES_PATH}/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(values),
  })

  return readAgency(response)
}

export async function deleteAgency(token: string, id: string): Promise<void> {
  await send(token, `${AGENCIES_PATH}/${id}`, { method: 'DELETE' })
}
