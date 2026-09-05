import { ApiError, apiFetch } from '../../../shared/api/client'
import { apiUrl } from '../../../shared/config/env'
import type { Archivo, ArchivoCategoria } from '../types'

const GENERIC_LIST_ERROR = 'No se pudieron cargar los archivos, intentá nuevamente.'
const GENERIC_UPLOAD_ERROR = 'No se pudo subir el archivo, intentá nuevamente.'
const GENERIC_DOWNLOAD_ERROR = 'No se pudo descargar el archivo, intentá nuevamente.'

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object'
}

function isArchivo(value: unknown): value is Archivo {
  return (
    isRecord(value) &&
    typeof value.id === 'string' &&
    (value.categoria === 'foto' || value.categoria === 'plano') &&
    typeof value.nombreOriginal === 'string' &&
    typeof value.mimeType === 'string' &&
    typeof value.hashSha256 === 'string' &&
    typeof value.fechaCreacion === 'string'
  )
}

function toArchivo(value: Record<string, unknown>): Archivo {
  return {
    id: value.id as string,
    categoria: value.categoria as ArchivoCategoria,
    nombreOriginal: value.nombreOriginal as string,
    mimeType: value.mimeType as string,
    hashSha256: value.hashSha256 as string,
    fechaCreacion: value.fechaCreacion as string,
  }
}

function isArchivoList(value: unknown): value is { archivos: unknown[] } {
  return isRecord(value) && Array.isArray(value.archivos)
}

function toArchivos(value: unknown, genericError: string): Archivo[] {
  if (!isArchivoList(value) || !value.archivos.every(isArchivo)) {
    throw new Error(genericError)
  }

  return value.archivos.map(toArchivo)
}

export async function listLoteoArchivos(
  loteoId: string,
  token: string,
  signal?: AbortSignal,
): Promise<Archivo[]> {
  let body: unknown
  try {
    body = await apiFetch<unknown>(`/api/v1/loteos/${encodeURIComponent(loteoId)}/archivos`, {
      token,
      signal,
    })
  } catch (error) {
    if (error instanceof ApiError || (error instanceof DOMException && error.name === 'AbortError')) {
      throw error
    }
    throw new Error(GENERIC_LIST_ERROR, { cause: error })
  }

  return toArchivos(body, GENERIC_LIST_ERROR)
}

export async function listLoteArchivos(
  loteoId: string,
  loteId: string,
  token: string,
  signal?: AbortSignal,
): Promise<Archivo[]> {
  let body: unknown
  try {
    body = await apiFetch<unknown>(
      `/api/v1/loteos/${encodeURIComponent(loteoId)}/lotes/${encodeURIComponent(loteId)}/archivos`,
      { token, signal },
    )
  } catch (error) {
    if (error instanceof ApiError || (error instanceof DOMException && error.name === 'AbortError')) {
      throw error
    }
    throw new Error(GENERIC_LIST_ERROR, { cause: error })
  }

  return toArchivos(body, GENERIC_LIST_ERROR)
}

function archivoForm(categoria: ArchivoCategoria, file: File): FormData {
  const form = new FormData()
  form.append('categoria', categoria)
  form.append('archivo', file, file.name)
  return form
}

export async function uploadLoteoArchivo(
  loteoId: string,
  categoria: ArchivoCategoria,
  file: File,
  token: string,
): Promise<Archivo> {
  let body: unknown
  try {
    body = await apiFetch<unknown>(`/api/v1/loteos/${encodeURIComponent(loteoId)}/archivos`, {
      method: 'POST',
      body: archivoForm(categoria, file),
      token,
    })
  } catch (error) {
    if (error instanceof ApiError) {
      throw error
    }
    throw new Error(GENERIC_UPLOAD_ERROR, { cause: error })
  }

  if (!isArchivo(body)) {
    throw new Error(GENERIC_UPLOAD_ERROR)
  }
  return toArchivo(body)
}

export async function uploadLoteArchivo(
  loteoId: string,
  loteId: string,
  categoria: ArchivoCategoria,
  file: File,
  token: string,
): Promise<Archivo> {
  let body: unknown
  try {
    body = await apiFetch<unknown>(
      `/api/v1/loteos/${encodeURIComponent(loteoId)}/lotes/${encodeURIComponent(loteId)}/archivos`,
      { method: 'POST', body: archivoForm(categoria, file), token },
    )
  } catch (error) {
    if (error instanceof ApiError) {
      throw error
    }
    throw new Error(GENERIC_UPLOAD_ERROR, { cause: error })
  }

  if (!isArchivo(body)) {
    throw new Error(GENERIC_UPLOAD_ERROR)
  }
  return toArchivo(body)
}

export async function deleteArchivo(loteoId: string, archivoId: string, token: string): Promise<void> {
  try {
    await apiFetch<void>(
      `/api/v1/loteos/${encodeURIComponent(loteoId)}/archivos/${encodeURIComponent(archivoId)}`,
      { method: 'DELETE', token },
    )
  } catch (error) {
    if (error instanceof ApiError) {
      throw error
    }
    throw new Error('No se pudo eliminar el archivo, intentá nuevamente.', { cause: error })
  }
}

// fetchArchivoContent bypasses apiFetch: the response is the file's own
// bytes, not JSON, so it can't go through apiFetch's JSON decoding. The
// caller turns the blob into an object URL — the API needs the caller's
// bearer token, so a plain <img src> can't reach it directly.
export async function fetchArchivoContent(
  loteoId: string,
  archivoId: string,
  token: string,
  signal?: AbortSignal,
): Promise<Blob> {
  let response: Response
  try {
    response = await fetch(
      `${apiUrl}/api/v1/loteos/${encodeURIComponent(loteoId)}/archivos/${encodeURIComponent(archivoId)}`,
      { headers: { Authorization: `Bearer ${token}` }, signal },
    )
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') {
      throw error
    }
    throw new ApiError(
      'No se pudo conectar con el servidor. Revisá tu conexión e intentá de nuevo.',
      'network_error',
      0,
    )
  }

  if (!response.ok) {
    let code = 'http_error'
    let message = GENERIC_DOWNLOAD_ERROR
    try {
      const data: unknown = await response.json()
      if (isRecord(data)) {
        if (typeof data.code === 'string') code = data.code
        if (typeof data.message === 'string') message = data.message
      }
    } catch {
      // Body wasn't JSON; the defaults above stand.
    }
    throw new ApiError(message, code, response.status)
  }

  return response.blob()
}
