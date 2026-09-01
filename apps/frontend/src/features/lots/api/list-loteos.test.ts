import { afterEach, describe, expect, it, vi } from 'vitest'
import { listLoteos } from './list-loteos'
import type { LoteoSummary } from '../types'

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

const GENERIC_ERROR = 'No se pudo cargar el listado de loteos, intentá nuevamente.'

const loteoResponse = {
  id: 'loteo-1',
  nombre: 'Loteo Las Acacias',
  ubicacion: 'Río Ceballos, Córdoba',
  descripcion: 'Loteo residencial sobre ruta E-53.',
  cantidadManzanas: 12,
  cantidadLotes: 148,
  cantidadCalles: 8,
  tienePlano: true,
  tieneDxf: true,
  fechaCreacion: '2026-01-10T12:00:00Z',
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('listLoteos', () => {
  it('sends the bearer token and returns the loteos', async () => {
    const fetchMock = stubFetch(jsonResponse(200, { loteos: [loteoResponse] }))

    const loteos = await listLoteos('token-123')

    expect(loteos).toEqual<LoteoSummary[]>([loteoResponse])

    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toContain('/api/v1/loteos')
    expect(url).not.toContain('?q=')
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer token-123')
  })

  it('adds the search text as an encoded q query param', async () => {
    const fetchMock = stubFetch(jsonResponse(200, { loteos: [] }))

    await listLoteos('token-123', { q: 'las acacias' })

    const [url] = fetchMock.mock.calls[0] as unknown as [string]
    expect(url).toContain('/api/v1/loteos?q=las%20acacias')
  })

  it('turns an absent or null descripcion into an empty string', async () => {
    const { descripcion: _drop, ...withoutDescripcion } = loteoResponse
    stubFetch(
      jsonResponse(200, {
        loteos: [withoutDescripcion, { ...loteoResponse, id: 'loteo-2', descripcion: null }],
      }),
    )

    const [first, second] = await listLoteos('token-123')

    expect(first.descripcion).toBe('')
    expect(second.descripcion).toBe('')
  })

  it('rejects a loteo whose descripcion is not a string', async () => {
    stubFetch(jsonResponse(200, { loteos: [{ ...loteoResponse, descripcion: { es: 'x' } }] }))

    await expect(listLoteos('token-123')).rejects.toThrow(GENERIC_ERROR)
  })

  it('returns an empty list when loteos is null', async () => {
    stubFetch(jsonResponse(200, { loteos: null }))

    await expect(listLoteos('token-123')).resolves.toEqual([])
  })

  it('rejects a 2xx body that does not carry the expected contract', async () => {
    stubFetch(jsonResponse(200, {}))

    await expect(listLoteos('token-123')).rejects.toThrow(GENERIC_ERROR)
  })

  it('rejects a 2xx body whose loteos are not summaries', async () => {
    stubFetch(jsonResponse(200, { loteos: [{ id: 'loteo-1' }] }))

    await expect(listLoteos('token-123')).rejects.toThrow(GENERIC_ERROR)
  })

  it('rejects a 2xx body that is not json', async () => {
    stubFetch(new Response('not json', { status: 200 }))

    await expect(listLoteos('token-123')).rejects.toThrow(GENERIC_ERROR)
  })

  it('forwards the abort signal to fetch', async () => {
    const fetchMock = stubFetch(jsonResponse(200, { loteos: [] }))
    const controller = new AbortController()

    await listLoteos('token-123', { signal: controller.signal })

    const [, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(init.signal).toBe(controller.signal)
  })

  it('throws the message returned by the backend', async () => {
    stubFetch(jsonResponse(403, { code: 'forbidden', message: 'No autorizado' }))

    await expect(listLoteos('token-123')).rejects.toThrow('No autorizado')
  })

  it('surfaces the shared client message when the server errors without json', async () => {
    stubFetch(new Response('boom', { status: 500 }))

    await expect(listLoteos('token-123')).rejects.toThrow(/error inesperado/i)
  })

  it('rethrows an AbortError instead of masking it as a listing failure', async () => {
    const fetchMock = vi.fn(async () => {
      throw new DOMException('aborted', 'AbortError')
    })
    vi.stubGlobal('fetch', fetchMock)

    await expect(listLoteos('token-123')).rejects.toThrow(/aborted/)
  })
})
