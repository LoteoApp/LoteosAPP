import { afterEach, describe, expect, it, vi } from 'vitest'
import { updateCalle } from './update-calle'

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
  nombre: 'Los Álamos',
  tipo: 'asfalto',
}

const calleResponse = {
  id: 'ca-1',
  nombre: 'Los Álamos',
  tipo: 'asfalto',
  poligono: [{ x: 0, y: 0 }, { x: 1, y: 0 }, { x: 1, y: 1 }],
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('updateCalle', () => {
  it('PATCHes the calle and returns the parsed body', async () => {
    const fetchMock = stubFetch(jsonResponse(200, calleResponse))

    const calle = await updateCalle('loteo-1', 'ca-1', payload, 'token-123')

    expect(calle).toMatchObject({ id: 'ca-1', nombre: 'Los Álamos', tipo: 'asfalto' })
    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toContain('/api/v1/loteos/loteo-1/calles/ca-1')
    expect(init.method).toBe('PATCH')
    expect(JSON.parse(String(init.body))).toEqual(payload)
  })

  it('rejects a 2xx body that is not a calle', async () => {
    stubFetch(jsonResponse(200, { id: 'ca-1' }))
    await expect(updateCalle('loteo-1', 'ca-1', payload, 'token')).rejects.toThrow(
      'No se pudo guardar la calle, intentá nuevamente.',
    )
  })

  it('preserves api errors from the server', async () => {
    stubFetch(jsonResponse(400, { code: 'invalid_calle_tipo', message: 'Tipo inválido' }))

    await expect(updateCalle('loteo-1', 'ca-1', payload, 'token')).rejects.toMatchObject({
      code: 'invalid_calle_tipo',
      message: 'Tipo inválido',
    })
  })

  it('normalizes an unexpected transport error', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('offline')))

    await expect(updateCalle('loteo-1', 'ca-1', payload, 'token')).rejects.toThrow(
      'No se pudo conectar con el servidor.',
    )
  })

  it('propagates an aborted request', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new DOMException('aborted', 'AbortError')))

    await expect(updateCalle('loteo-1', 'ca-1', payload, 'token')).rejects.toMatchObject({
      name: 'AbortError',
    })
  })
})
