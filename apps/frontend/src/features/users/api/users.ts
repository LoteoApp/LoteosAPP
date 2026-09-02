import { apiFetch } from '../../../shared/api/client'
import type { Usuario, UsuarioFormValues, UsuarioUpdateValues } from '../types'

const USERS_PATH = '/api/v1/usuarios'
const GENERIC_ERROR = 'No se pudo completar la operación, intentá nuevamente.'

function isUsuarioResponse(value: unknown): value is Usuario {
  if (value === null || typeof value !== 'object') {
    return false
  }

  const candidate = value as Record<string, unknown>
  return (
    typeof candidate.id === 'string' &&
    typeof candidate.email === 'string' &&
    typeof candidate.nombre === 'string' &&
    typeof candidate.apellido === 'string' &&
    typeof candidate.rol === 'string'
  )
}

export async function listUsers(token: string, signal?: AbortSignal): Promise<Usuario[]> {
  const body = await apiFetch<unknown>(`${USERS_PATH}?incluirBajas=true`, { token, signal })

  if (body === null || typeof body !== 'object' || !('usuarios' in body)) {
    throw new Error(GENERIC_ERROR)
  }

  const { usuarios } = body as { usuarios?: unknown }
  if (usuarios === null || usuarios === undefined) {
    return []
  }
  if (!Array.isArray(usuarios) || !usuarios.every(isUsuarioResponse)) {
    throw new Error(GENERIC_ERROR)
  }

  return usuarios
}

export type CreatedUsuario = {
  usuario: Usuario
  temporaryPassword: string
}

export async function createUser(token: string, values: UsuarioFormValues): Promise<CreatedUsuario> {
  const body = await apiFetch<unknown>(USERS_PATH, { method: 'POST', body: values, token })

  if (!isUsuarioResponse(body)) {
    throw new Error(GENERIC_ERROR)
  }
  const { temporaryPassword } = body as { temporaryPassword?: unknown }
  if (typeof temporaryPassword !== 'string' || temporaryPassword === '') {
    throw new Error(GENERIC_ERROR)
  }

  return { usuario: body, temporaryPassword }
}

export async function updateUser(
  token: string,
  id: string,
  values: UsuarioUpdateValues,
): Promise<Usuario> {
  const body = await apiFetch<unknown>(`${USERS_PATH}/${id}`, {
    method: 'PATCH',
    body: values,
    token,
  })

  if (!isUsuarioResponse(body)) {
    throw new Error(GENERIC_ERROR)
  }

  return body
}

export async function deactivateUser(token: string, id: string): Promise<void> {
  await apiFetch(`${USERS_PATH}/${id}`, { method: 'DELETE', token })
}

export async function reactivateUser(token: string, id: string): Promise<Usuario> {
  const body = await apiFetch<unknown>(`${USERS_PATH}/${id}/reactivar`, { method: 'POST', token })

  if (!isUsuarioResponse(body)) {
    throw new Error(GENERIC_ERROR)
  }

  return body
}
