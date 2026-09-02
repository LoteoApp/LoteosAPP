import { afterEach, describe, expect, it, vi } from 'vitest'
import { getLoteo } from './get-loteo'

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

const GENERIC_ERROR = 'No se pudo cargar el loteo, intentá nuevamente.'

const detailResponse = {
  id: 'loteo-1',
  nombre: 'Las Acacias',
  ubicacion: 'Río Ceballos, Córdoba',
  descripcion: 'Sobre ruta E-53.',
  contorno: [
    { x: 0, y: 0 },
    { x: 10, y: 0 },
    { x: 10, y: 10 },
  ],
  manzanas: [{ id: 'mz-1', numero: '1', poligono: [{ x: 1, y: 1 }, { x: 2, y: 1 }, { x: 2, y: 2 }] }],
  lotes: [
    {
      id: 'lt-1',
      manzanaId: 'mz-1',
      numero: '7',
      precio: 150000,
      moneda: 'USD',
      superficie: 300,
      caracteristicas: 'Esquina',
      poligono: [{ x: 1, y: 1 }, { x: 1.5, y: 1 }, { x: 1.5, y: 1.5 }],
    },
  ],
  calles: [{ id: 'ca-1', nombre: 'Los Álamos', tipo: 'asfalto', poligono: [] }],
  fechaCreacion: '2026-08-20T12:00:00Z',
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('getLoteo', () => {
  it('sends the bearer token and returns the parsed detail', async () => {
    const fetchMock = stubFetch(jsonResponse(200, detailResponse))

    const loteo = await getLoteo('loteo-1', 'token-123')

    expect(loteo.id).toBe('loteo-1')
    expect(loteo.contorno).toHaveLength(3)
    expect(loteo.manzanas[0]).toMatchObject({ id: 'mz-1', numero: '1' })
    expect(loteo.lotes[0]).toMatchObject({ precio: 150000, moneda: 'USD', superficie: 300 })
    expect(loteo.calles[0].poligono).toEqual([])

    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toContain('/api/v1/loteos/loteo-1')
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer token-123')
  })

  it('url-encodes the loteo id in the path', async () => {
    const fetchMock = stubFetch(jsonResponse(200, detailResponse))

    await getLoteo('a/b c', 'token-123')

    const [url] = fetchMock.mock.calls[0] as unknown as [string]
    expect(url).toContain('/api/v1/loteos/a%2Fb%20c')
  })

  it('normalizes a missing polygon to an empty array', async () => {
    const { poligono: _drop, ...manzanaWithoutPolygon } = detailResponse.manzanas[0]
    stubFetch(
      jsonResponse(200, { ...detailResponse, contorno: undefined, manzanas: [manzanaWithoutPolygon] }),
    )

    const loteo = await getLoteo('loteo-1', 'token-123')

    expect(loteo.contorno).toEqual([])
    expect(loteo.manzanas[0].poligono).toEqual([])
  })

  it('keeps a null precio and superficie as null', async () => {
    stubFetch(
      jsonResponse(200, {
        ...detailResponse,
        lotes: [{ ...detailResponse.lotes[0], precio: null, superficie: null }],
      }),
    )

    const loteo = await getLoteo('loteo-1', 'token-123')

    expect(loteo.lotes[0].precio).toBeNull()
    expect(loteo.lotes[0].superficie).toBeNull()
  })

  it('rejects a 2xx body that does not carry the expected contract', async () => {
    stubFetch(jsonResponse(200, { id: 'loteo-1', nombre: 'x' }))

    await expect(getLoteo('loteo-1', 'token-123')).rejects.toThrow(GENERIC_ERROR)
  })

  it('rejects a 2xx body that is not json', async () => {
    stubFetch(new Response('not json', { status: 200 }))

    await expect(getLoteo('loteo-1', 'token-123')).rejects.toThrow(GENERIC_ERROR)
  })

  it('rethrows the ApiError for a 404 so the hook can map it', async () => {
    stubFetch(jsonResponse(404, { code: 'loteo_not_found', message: 'El loteo solicitado no existe' }))

    await expect(getLoteo('missing', 'token-123')).rejects.toMatchObject({
      name: 'ApiError',
      status: 404,
      code: 'loteo_not_found',
    })
  })

  it('forwards the abort signal and rethrows an AbortError untouched', async () => {
    const fetchMock = vi.fn(async () => {
      throw new DOMException('aborted', 'AbortError')
    })
    vi.stubGlobal('fetch', fetchMock)
    const controller = new AbortController()

    await expect(
      getLoteo('loteo-1', 'token-123', { signal: controller.signal }),
    ).rejects.toThrow(/aborted/)
  })
})
