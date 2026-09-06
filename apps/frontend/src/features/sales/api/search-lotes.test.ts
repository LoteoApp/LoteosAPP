import { afterEach, describe, expect, it, vi } from 'vitest'
import { searchLotes } from './search-lotes'

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

const GENERIC_ERROR = 'No se pudieron cargar los lotes, intentá nuevamente.'

const loteResponse = {
  id: 'lt-1',
  numero: '7',
  manzanaId: 'mz-1',
  manzanaNumero: '1',
  loteoId: 'loteo-1',
  loteoNombre: 'Norte',
  estado: 'disponible',
  precio: 150000,
  moneda: 'USD',
  superficie: 300,
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('searchLotes', () => {
  it('returns the parsed lotes', async () => {
    const fetchMock = stubFetch(jsonResponse(200, { lotes: [loteResponse] }))

    const lotes = await searchLotes('token-123')

    expect(lotes).toEqual([loteResponse])
    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toContain('/api/v1/lotes')
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer token-123')
  })

  it('sends the search and state filters as query params', async () => {
    const fetchMock = stubFetch(jsonResponse(200, { lotes: [] }))

    await searchLotes('token-123', { search: 'norte', estado: 'disponible' })

    const [url] = fetchMock.mock.calls[0] as unknown as [string]
    expect(url).toContain('q=norte')
    expect(url).toContain('estado=disponible')
  })

  it('omits the query string when no filter is given', async () => {
    const fetchMock = stubFetch(jsonResponse(200, { lotes: [] }))

    await searchLotes('token-123')

    const [url] = fetchMock.mock.calls[0] as unknown as [string]
    expect(url).not.toContain('?')
  })

  it('normalizes a null lotes list to an empty array', async () => {
    stubFetch(jsonResponse(200, { lotes: null }))

    await expect(searchLotes('token-123')).resolves.toEqual([])
  })

  it('leaves a missing price, area, number or currency empty', async () => {
    const { precio: _p, superficie: _s, numero: _n, moneda: _m, ...partial } = loteResponse
    stubFetch(jsonResponse(200, { lotes: [partial] }))

    const [lote] = await searchLotes('token-123')

    expect(lote.precio).toBeNull()
    expect(lote.superficie).toBeNull()
    expect(lote.numero).toBe('')
    expect(lote.moneda).toBe('')
  })

  it('rejects a body that does not carry a lotes list', async () => {
    stubFetch(jsonResponse(200, { otros: [] }))

    await expect(searchLotes('token-123')).rejects.toThrow(GENERIC_ERROR)
  })

  it('rejects a lote whose estado is not a known state', async () => {
    stubFetch(jsonResponse(200, { lotes: [{ ...loteResponse, estado: 'rifado' }] }))

    await expect(searchLotes('token-123')).rejects.toThrow(GENERIC_ERROR)
  })

  it('rethrows an ApiError so the caller can map the code', async () => {
    stubFetch(jsonResponse(403, { code: 'forbidden', message: 'No autorizado' }))

    await expect(searchLotes('token-123')).rejects.toMatchObject({
      name: 'ApiError',
      status: 403,
      code: 'forbidden',
    })
  })
})
