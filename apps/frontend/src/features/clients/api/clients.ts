import { apiUrl } from '../../../shared/config/env'
import type { Cliente, ClienteFormValues } from '../types'

const CLIENTS_PATH = '/api/v1/clientes'
const GENERIC_ERROR = 'No se pudo completar la operación, intentá nuevamente.'

type ClientResponse = Omit<Cliente, 'celular' | 'email'> & {
  celular?: string | null
  email?: string | null
}

function isClientResponse(value: unknown): value is ClientResponse {
  if (value === null || typeof value !== 'object') {
    return false
  }

  const candidate = value as Record<string, unknown>
  return (
    typeof candidate.id === 'string' &&
    typeof candidate.nombre === 'string' &&
    typeof candidate.apellido === 'string' &&
    typeof candidate.dni === 'string'
  )
}

function toClient(raw: ClientResponse): Cliente {
  return {
    id: raw.id,
    nombre: raw.nombre,
    apellido: raw.apellido,
    dni: raw.dni,
    celular: raw.celular ?? '',
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

async function readClient(response: Response): Promise<Cliente> {
  const body = await readJSON(response)
  if (!isClientResponse(body)) {
    throw new Error(GENERIC_ERROR)
  }

  return toClient(body)
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

export async function listClients(token: string, signal?: AbortSignal): Promise<Cliente[]> {
  const response = await send(token, CLIENTS_PATH, { signal })
  const body = await readJSON(response)

  if (body === null || typeof body !== 'object' || !('clientes' in body)) {
    throw new Error(GENERIC_ERROR)
  }

  const { clientes } = body as { clientes?: unknown }
  if (clientes === null || clientes === undefined) {
    return []
  }
  if (!Array.isArray(clientes) || !clientes.every(isClientResponse)) {
    throw new Error(GENERIC_ERROR)
  }

  return clientes.map(toClient)
}

export async function createClient(token: string, values: ClienteFormValues): Promise<Cliente> {
  const response = await send(token, CLIENTS_PATH, {
    method: 'POST',
    body: JSON.stringify(values),
  })

  return readClient(response)
}

export async function updateClient(
  token: string,
  id: string,
  values: ClienteFormValues,
): Promise<Cliente> {
  const response = await send(token, `${CLIENTS_PATH}/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(values),
  })

  return readClient(response)
}

export async function deleteClient(token: string, id: string): Promise<void> {
  await send(token, `${CLIENTS_PATH}/${id}`, { method: 'DELETE' })
}
