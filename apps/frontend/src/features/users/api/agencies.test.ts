import { afterEach, describe, expect, it, vi } from 'vitest'
import { listAgencyOptions } from './agencies'

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

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('listAgencyOptions', () => {
  it('sends the bearer token and keeps only what the dropdown needs', async () => {
    const fetchMock = stubFetch(
      jsonResponse(200, {
        inmobiliarias: [
          { id: 'inm-1', razonSocial: 'Lotes del Sur', cuit: '30712345678', telefono: null },
          { id: 'inm-2', razonSocial: 'Altamira' },
        ],
      })
    )

    await expect(listAgencyOptions('token-123')).resolves.toEqual([
      { id: 'inm-1', razonSocial: 'Lotes del Sur' },
      { id: 'inm-2', razonSocial: 'Altamira' },
    ])

    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toContain('/api/v1/inmobiliarias')
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer token-123')
  })

  it('forwards the abort signal', async () => {
    const fetchMock = stubFetch(jsonResponse(200, { inmobiliarias: [] }))
    const controller = new AbortController()

    await listAgencyOptions('token-123', controller.signal)

    const [, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(init.signal).toBe(controller.signal)
  })

  it('treats a null list as no agencies', async () => {
    stubFetch(jsonResponse(200, { inmobiliarias: null }))

    await expect(listAgencyOptions('token-123')).resolves.toEqual([])
  })

  it('rejects a payload without the inmobiliarias key', async () => {
    stubFetch(jsonResponse(200, { otros: [] }))

    await expect(listAgencyOptions('token-123')).rejects.toThrow(
      'No se pudieron cargar las inmobiliarias, intentá nuevamente.'
    )
  })

  it('rejects a list whose items are not agencies', async () => {
    stubFetch(jsonResponse(200, { inmobiliarias: [{ id: 'inm-1' }] }))

    await expect(listAgencyOptions('token-123')).rejects.toThrow(
      'No se pudieron cargar las inmobiliarias, intentá nuevamente.'
    )
  })

  it('throws the message returned by the backend', async () => {
    stubFetch(jsonResponse(403, { code: 'forbidden', message: 'No tenés permisos para esta acción' }))

    await expect(listAgencyOptions('token-123')).rejects.toThrow('No tenés permisos para esta acción')
  })
})
