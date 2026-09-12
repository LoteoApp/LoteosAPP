import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '../../../shared/api/client'
import { apiUrl } from '../../../shared/config/env'
import * as client from '../../../shared/api/client'
import {
  deleteAttachment,
  fetchAttachmentContent,
  listLoteAttachments,
  listLoteoAttachments,
  uploadLoteAttachment,
  uploadLoteoAttachment,
} from './archivos'

const attachmentPayload = {
  id: 'archivo-1',
  categoria: 'foto',
  nombreOriginal: 'foto.jpg',
  mimeType: 'image/jpeg',
  hashSha256: 'abc123',
  fechaCreacion: '2026-01-01T00:00:00Z',
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('listLoteoAttachments', () => {
  it('fetches and unwraps the archivos list', async () => {
    const apiFetch = vi.spyOn(client, 'apiFetch').mockResolvedValue({ archivos: [attachmentPayload] })

    const result = await listLoteoAttachments('loteo 1', 'tok')

    expect(apiFetch).toHaveBeenCalledWith('/api/v1/loteos/loteo%201/archivos', {
      token: 'tok',
      signal: undefined,
    })
    expect(result).toEqual([
      {
        id: 'archivo-1',
        category: 'foto',
        nombreOriginal: 'foto.jpg',
        mimeType: 'image/jpeg',
        hashSha256: 'abc123',
        fechaCreacion: '2026-01-01T00:00:00Z',
      },
    ])
  })

  it('throws when the response is malformed', async () => {
    vi.spyOn(client, 'apiFetch').mockResolvedValue({ archivos: [{ id: 'x' }] })

    await expect(listLoteoAttachments('loteo-1', 'tok')).rejects.toThrow(
      'No se pudieron cargar los archivos, intentá nuevamente.',
    )
  })

  it('rethrows an ApiError as-is', async () => {
    const apiError = new ApiError('no autorizado', 'forbidden', 403)
    vi.spyOn(client, 'apiFetch').mockRejectedValue(apiError)

    await expect(listLoteoAttachments('loteo-1', 'tok')).rejects.toBe(apiError)
  })
})

describe('listLoteAttachments', () => {
  it('fetches the archivos scoped to a lote', async () => {
    const apiFetch = vi.spyOn(client, 'apiFetch').mockResolvedValue({ archivos: [] })

    await listLoteAttachments('loteo-1', 'lote-1', 'tok')

    expect(apiFetch).toHaveBeenCalledWith('/api/v1/loteos/loteo-1/lotes/lote-1/archivos', {
      token: 'tok',
      signal: undefined,
    })
  })

  it('rethrows an ApiError as-is', async () => {
    const apiError = new ApiError('no encontrado', 'loteo_not_found', 404)
    vi.spyOn(client, 'apiFetch').mockRejectedValue(apiError)

    await expect(listLoteAttachments('loteo-1', 'lote-1', 'tok')).rejects.toBe(apiError)
  })
})

describe('uploadLoteoAttachment', () => {
  it('POSTs the file and categoria as multipart form data', async () => {
    const apiFetch = vi.spyOn(client, 'apiFetch').mockResolvedValue(attachmentPayload)
    const file = new File(['bytes'], 'foto.jpg', { type: 'image/jpeg' })

    const result = await uploadLoteoAttachment('loteo-1', 'foto', file, 'tok')

    expect(apiFetch).toHaveBeenCalledTimes(1)
    const [path, options] = apiFetch.mock.calls[0]
    expect(path).toBe('/api/v1/loteos/loteo-1/archivos')
    expect(options?.method).toBe('POST')
    expect(options?.token).toBe('tok')
    const body = options?.body as FormData
    expect(body.get('categoria')).toBe('foto')
    expect((body.get('archivo') as File).name).toBe('foto.jpg')
    expect(result.id).toBe('archivo-1')
  })

  it('wraps a non-ApiError failure with a generic message', async () => {
    vi.spyOn(client, 'apiFetch').mockRejectedValue(new Error('boom'))
    const file = new File(['bytes'], 'foto.jpg', { type: 'image/jpeg' })

    await expect(uploadLoteoAttachment('loteo-1', 'foto', file, 'tok')).rejects.toThrow(
      'No se pudo subir el archivo, intentá nuevamente.',
    )
  })

  it('rethrows an ApiError as-is', async () => {
    const apiError = new ApiError('archivo inválido', 'invalid_archivo', 400)
    vi.spyOn(client, 'apiFetch').mockRejectedValue(apiError)
    const file = new File(['bytes'], 'foto.jpg', { type: 'image/jpeg' })

    await expect(uploadLoteoAttachment('loteo-1', 'foto', file, 'tok')).rejects.toBe(apiError)
  })

  it('throws when the response body is not an attachment', async () => {
    vi.spyOn(client, 'apiFetch').mockResolvedValue({ id: 'x' })
    const file = new File(['bytes'], 'foto.jpg', { type: 'image/jpeg' })

    await expect(uploadLoteoAttachment('loteo-1', 'foto', file, 'tok')).rejects.toThrow(
      'No se pudo subir el archivo, intentá nuevamente.',
    )
  })
})

describe('uploadLoteAttachment', () => {
  it('POSTs to the lote-scoped endpoint', async () => {
    const apiFetch = vi.spyOn(client, 'apiFetch').mockResolvedValue({ ...attachmentPayload, categoria: 'plano' })
    const file = new File(['bytes'], 'plano.pdf', { type: 'application/pdf' })

    await uploadLoteAttachment('loteo-1', 'lote-1', 'plano', file, 'tok')

    const [path] = apiFetch.mock.calls[0]
    expect(path).toBe('/api/v1/loteos/loteo-1/lotes/lote-1/archivos')
  })

  it('rethrows an ApiError as-is', async () => {
    const apiError = new ApiError('lote inválido', 'lote_not_found', 404)
    vi.spyOn(client, 'apiFetch').mockRejectedValue(apiError)
    const file = new File(['bytes'], 'plano.pdf', { type: 'application/pdf' })

    await expect(uploadLoteAttachment('loteo-1', 'lote-1', 'plano', file, 'tok')).rejects.toBe(apiError)
  })
})

describe('deleteAttachment', () => {
  it('DELETEs the archivo', async () => {
    const apiFetch = vi.spyOn(client, 'apiFetch').mockResolvedValue(undefined)

    await deleteAttachment('loteo-1', 'archivo-1', 'tok')

    expect(apiFetch).toHaveBeenCalledWith('/api/v1/loteos/loteo-1/archivos/archivo-1', {
      method: 'DELETE',
      token: 'tok',
    })
  })

  it('wraps a non-ApiError failure', async () => {
    vi.spyOn(client, 'apiFetch').mockRejectedValue(new Error('boom'))

    await expect(deleteAttachment('loteo-1', 'archivo-1', 'tok')).rejects.toThrow(
      'No se pudo eliminar el archivo, intentá nuevamente.',
    )
  })

  it('rethrows an ApiError as-is', async () => {
    const apiError = new ApiError('no encontrado', 'archivo_not_found', 404)
    vi.spyOn(client, 'apiFetch').mockRejectedValue(apiError)

    await expect(deleteAttachment('loteo-1', 'archivo-1', 'tok')).rejects.toBe(apiError)
  })
})

describe('fetchAttachmentContent', () => {
  function stubFetch(response: Response) {
    const mock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) => response)
    vi.stubGlobal('fetch', mock)
    return mock
  }

  it('fetches the raw bytes with the bearer token', async () => {
    const mock = stubFetch(new Response('hola', { status: 200, headers: { 'Content-Type': 'image/jpeg' } }))

    const result = await fetchAttachmentContent('loteo-1', 'archivo-1', 'tok')

    const [url, init] = mock.mock.calls[0]
    expect(url).toBe(`${apiUrl}/api/v1/loteos/loteo-1/archivos/archivo-1`)
    expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer tok')
    expect(await result.text()).toBe('hola')
  })

  it('maps a backend error response to an ApiError', async () => {
    stubFetch(new Response(JSON.stringify({ code: 'archivo_not_found', message: 'no existe' }), { status: 404 }))

    await expect(fetchAttachmentContent('loteo-1', 'archivo-1', 'tok')).rejects.toMatchObject({
      code: 'archivo_not_found',
      message: 'no existe',
      status: 404,
    })
  })

  it('maps a network failure to an ApiError', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => {
        throw new TypeError('network down')
      }),
    )

    await expect(fetchAttachmentContent('loteo-1', 'archivo-1', 'tok')).rejects.toMatchObject({
      code: 'network_error',
    })
  })

  it('lets an AbortError propagate instead of wrapping it', async () => {
    const abortError = new DOMException('aborted', 'AbortError')
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => {
        throw abortError
      }),
    )

    await expect(fetchAttachmentContent('loteo-1', 'archivo-1', 'tok')).rejects.toBe(abortError)
  })
})
