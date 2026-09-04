import { afterEach, describe, expect, it, vi } from 'vitest'
import { updateLote } from './update-lote'
import type { UpdateLotePayload } from '../lib/loteFormValues'

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function stubFetch(response: Response) {
  const fetchMock = vi.fn(async () => response)
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

const GENERIC_ERROR = 'No se pudo guardar el lote, intentá nuevamente.'

const payload: UpdateLotePayload = {
  numero: '12',
  precio: 4500000.5,
  moneda: 'ARS',
  superficie: 320.75,
  caracteristicas: 'Esquina',
}

const loteResponse = {
  id: 'lt-1',
  manzanaId: 'mz-1',
  numero: '12',
  precio: 4500000.5,
  moneda: 'ARS',
  superficie: 320.75,
  caracteristicas: 'Esquina',
  poligono: [
    { x: 0, y: 0 },
    { x: 1, y: 0 },
    { x: 1, y: 1 },
  ],
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('updateLote', () => {
  it('PATCHes the lote and returns the parsed body', async () => {
    const fetchMock = stubFetch(jsonResponse(200, loteResponse))

    const lote = await updateLote('loteo-1', 'lt-1', payload, 'token-123')

    expect(lote).toMatchObject({ id: 'lt-1', numero: '12', precio: 4500000.5 })
    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toContain('/api/v1/loteos/loteo-1/lotes/lt-1')
    expect(init.method).toBe('PATCH')
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer token-123')
    expect(JSON.parse(String(init.body))).toEqual(payload)
  })

  it('url-encodes both ids', async () => {
    const fetchMock = stubFetch(jsonResponse(200, loteResponse))

    await updateLote('a/b', 'c d', payload, 'token-123')

    const [url] = fetchMock.mock.calls[0] as unknown as [string]
    expect(url).toContain('/api/v1/loteos/a%2Fb/lotes/c%20d')
  })

  it('normalizes a missing polygon to an empty array', async () => {
    const { poligono: _drop, ...withoutPolygon } = loteResponse
    stubFetch(jsonResponse(200, withoutPolygon))

    const lote = await updateLote('loteo-1', 'lt-1', payload, 'token-123')

    expect(lote.poligono).toEqual([])
  })

  it('rejects a 2xx body that does not carry a lote', async () => {
    stubFetch(jsonResponse(200, { id: 'lt-1' }))

    await expect(updateLote('loteo-1', 'lt-1', payload, 'token-123')).rejects.toThrow(
      GENERIC_ERROR,
    )
  })

  it('rethrows an ApiError so the hook can map the code', async () => {
    stubFetch(
      jsonResponse(409, {
        code: 'lote_numero_in_use',
        message: 'Ya existe un lote con ese número en este loteo',
      }),
    )

    await expect(updateLote('loteo-1', 'lt-1', payload, 'token-123')).rejects.toMatchObject({
      name: 'ApiError',
      status: 409,
      code: 'lote_numero_in_use',
    })
  })

  it('wraps an unexpected failure in a Spanish error', async () => {
    stubFetch(new Response('not json', { status: 200 }))

    await expect(updateLote('loteo-1', 'lt-1', payload, 'token-123')).rejects.toThrow(
      GENERIC_ERROR,
    )
  })

  it('rethrows an AbortError untouched', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => {
        throw new DOMException('aborted', 'AbortError')
      }),
    )

    await expect(updateLote('loteo-1', 'lt-1', payload, 'token-123')).rejects.toThrow(/aborted/)
  })
})
