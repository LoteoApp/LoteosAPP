import { apiUrl } from '../config/env'

const NETWORK_ERROR_MESSAGE =
  'No se pudo conectar con el servidor. Revisá tu conexión e intentá de nuevo.'
const UNEXPECTED_ERROR_MESSAGE = 'Ocurrió un error inesperado. Intentá de nuevo en unos minutos.'

// ApiError carries the backend's own `code` and `message` (see
// response.ErrorResponse in the Go API) so callers can show the message as-is
// and branch on the code. `status` is 0 when the request never reached the
// server.
export class ApiError extends Error {
  readonly code: string
  readonly status: number

  constructor(message: string, code: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.status = status
  }
}

type ApiFetchOptions = {
  method?: string
  body?: unknown
  token?: string | null
  signal?: AbortSignal
}

export async function apiFetch<T = unknown>(
  path: string,
  { method = 'GET', body, token, signal }: ApiFetchOptions = {},
): Promise<T> {
  const headers: Record<string, string> = {}
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }

  let payload: BodyInit | undefined
  if (body instanceof FormData) {
    // Let the browser set multipart/form-data with its boundary.
    payload = body
  } else if (body !== undefined) {
    payload = JSON.stringify(body)
    headers['Content-Type'] = 'application/json'
  }

  let response: Response
  try {
    response = await fetch(`${apiUrl}${path}`, { method, headers, body: payload, signal })
  } catch (error) {
    // A caller that aborted the request wants the AbortError to propagate, not
    // a "network is down" message it would then have to filter back out.
    if (error instanceof DOMException && error.name === 'AbortError') {
      throw error
    }
    throw new ApiError(NETWORK_ERROR_MESSAGE, 'network_error', 0)
  }

  if (!response.ok) {
    throw await readError(response)
  }

  if (response.status === 204) {
    return undefined as T
  }

  const text = await response.text()
  return (text ? JSON.parse(text) : undefined) as T
}

async function readError(response: Response): Promise<ApiError> {
  let code = 'http_error'
  let message =
    response.status >= 500 ? UNEXPECTED_ERROR_MESSAGE : 'La solicitud no pudo completarse.'

  try {
    const data: unknown = await response.json()
    if (data && typeof data === 'object') {
      const shape = data as { code?: unknown; message?: unknown }
      if (typeof shape.code === 'string') code = shape.code
      if (typeof shape.message === 'string') message = shape.message
    }
  } catch {
    // Body wasn't JSON; the status-based message stands.
  }

  return new ApiError(message, code, response.status)
}

export function messageFromError(error: unknown): string {
  return error instanceof ApiError ? error.message : UNEXPECTED_ERROR_MESSAGE
}
