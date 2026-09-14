import { ApiError, apiFetch } from '../../../shared/api/client'
import { isAgencyListItem, listAgencies as listAgencyItems, type AgencyListItem } from '../../../shared/api/agencies'
import type { Agency, AgencyFormValues } from '../types'

const AGENCIES_PATH = '/api/v1/inmobiliarias'
const GENERIC_ERROR = 'No se pudo completar la operación, intentá nuevamente.'

function toAgency(raw: AgencyListItem): Agency {
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

function readAgency(body: unknown): Agency {
  if (!isAgencyListItem(body)) {
    throw new Error(GENERIC_ERROR)
  }

  return toAgency(body)
}

export async function listAgencies(
  token: string,
  signal?: AbortSignal,
): Promise<Agency[]> {
  return (await listAgencyItems(token, signal)).map(toAgency)
}

export async function createAgency(
  token: string,
  values: AgencyFormValues,
): Promise<Agency> {
  return readAgency(await request(token, AGENCIES_PATH, { method: 'POST', body: values }))
}

export async function updateAgency(
  token: string,
  id: string,
  values: AgencyFormValues,
): Promise<Agency> {
  return readAgency(
    await request(token, `${AGENCIES_PATH}/${id}`, { method: 'PATCH', body: values }),
  )
}

export async function deleteAgency(token: string, id: string): Promise<void> {
  await request(token, `${AGENCIES_PATH}/${id}`, { method: 'DELETE' })
}
