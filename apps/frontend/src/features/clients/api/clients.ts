import { apiUrl } from '../../../shared/config/env'
import type { Cliente, ClienteFormValues } from '../types'

const CLIENTS_PATH = '/api/v1/clientes'
const GENERIC_ERROR = 'No se pudo completar la operación, intentá nuevamente.'

// celular and email travel as optional strings: domain.Cliente serializes
// them with omitempty, so an absent value arrives as undefined rather than
// as an empty string.
type ClienteResponse = Omit<Cliente, 'celular' | 'email'> & {
  celular?: string | null
  email?: string | null
}

function toCliente(raw: ClienteResponse): Cliente {
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

export async function listClients(token: string): Promise<Cliente[]> {
  const response = await send(token, CLIENTS_PATH)
  const body: { clientes?: ClienteResponse[] } = await response.json()

  return (body.clientes ?? []).map(toCliente)
}

export async function createClient(token: string, values: ClienteFormValues): Promise<Cliente> {
  const response = await send(token, CLIENTS_PATH, {
    method: 'POST',
    body: JSON.stringify(values),
  })

  return toCliente((await response.json()) as ClienteResponse)
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

  return toCliente((await response.json()) as ClienteResponse)
}

export async function deleteClient(token: string, id: string): Promise<void> {
  await send(token, `${CLIENTS_PATH}/${id}`, { method: 'DELETE' })
}
