import { afterEach, describe, expect, it, vi } from 'vitest'
import { listAgencies } from './agencies'

const GENERIC_ERROR = 'No se pudo completar la operación, intentá nuevamente.'

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

describe('listAgencies', () => {
  it('returns the agencies from the backend', async () => {
    stubFetch(
      jsonResponse(200, {
        inmobiliarias: [
          { id: 'inm-1', razonSocial: 'Lotes del Sur' },
          { id: 'inm-2', razonSocial: 'Altamira' },
        ],
      }),
    )

    await expect(listAgencies('token-123')).resolves.toEqual([
      { id: 'inm-1', razonSocial: 'Lotes del Sur' },
      { id: 'inm-2', razonSocial: 'Altamira' },
    ])
  })

  it('returns an empty array when the backend sends no agencies', async () => {
    stubFetch(jsonResponse(200, { inmobiliarias: null }))

    await expect(listAgencies('token-123')).resolves.toEqual([])
  })

  it('rejects a response without an inmobiliarias field', async () => {
    stubFetch(jsonResponse(200, {}))

    await expect(listAgencies('token-123')).rejects.toThrow(GENERIC_ERROR)
  })

  it('rejects a malformed agency in the list', async () => {
    stubFetch(jsonResponse(200, { inmobiliarias: [{ id: 'inm-1' }] }))

    await expect(listAgencies('token-123')).rejects.toThrow(GENERIC_ERROR)
  })
})
