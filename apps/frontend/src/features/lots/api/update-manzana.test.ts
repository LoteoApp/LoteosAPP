import { afterEach, describe, expect, it, vi } from 'vitest'
import { updateManzana } from './update-manzana'

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

const payload = {
  numero: 'A',
  tieneAgua: true,
  tieneCloaca: false,
  tieneLuz: true,
  tieneGas: false,
  calleIds: ['ca-1'],
}

const manzanaResponse = {
  id: 'mz-1',
  numero: 'A',
  tieneAgua: true,
  tieneCloaca: false,
  tieneLuz: true,
  tieneGas: false,
  calleIds: ['ca-1'],
  poligono: [{ x: 0, y: 0 }, { x: 1, y: 0 }, { x: 1, y: 1 }],
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('updateManzana', () => {
  it('PATCHes the manzana and returns the parsed body', async () => {
    const fetchMock = stubFetch(jsonResponse(200, manzanaResponse))

    const manzana = await updateManzana('loteo-1', 'mz-1', payload, 'token-123')

    expect(manzana).toMatchObject({ id: 'mz-1', numero: 'A', tieneAgua: true, calleIds: ['ca-1'] })
    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toContain('/api/v1/loteos/loteo-1/manzanas/mz-1')
    expect(init.method).toBe('PATCH')
    expect(JSON.parse(String(init.body))).toEqual(payload)
  })

  it('rejects a 2xx body that is not a manzana', async () => {
    stubFetch(jsonResponse(200, { id: 'mz-1' }))
    await expect(updateManzana('loteo-1', 'mz-1', payload, 'token')).rejects.toThrow(
      'No se pudo guardar la manzana, intentá nuevamente.',
    )
  })

  it('preserves api errors from the server', async () => {
    stubFetch(jsonResponse(400, { code: 'unknown_calle', message: 'Calle inválida' }))

    await expect(updateManzana('loteo-1', 'mz-1', payload, 'token')).rejects.toMatchObject({
      code: 'unknown_calle',
      message: 'Calle inválida',
    })
  })

  it('normalizes an unexpected transport error', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('offline')))

    await expect(updateManzana('loteo-1', 'mz-1', payload, 'token')).rejects.toThrow(
      'No se pudo conectar con el servidor.',
    )
  })

  it('propagates an aborted request', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new DOMException('aborted', 'AbortError')))

    await expect(updateManzana('loteo-1', 'mz-1', payload, 'token')).rejects.toMatchObject({
      name: 'AbortError',
    })
  })
})
