import { afterEach, describe, expect, it, vi } from 'vitest'
import { createAgency, deleteAgency, listAgencies, updateAgency } from './agencies'
import { ApiError } from '../../../shared/api/client'

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
  it('maps the API payload, defaulting the optional fields to empty strings', async () => {
    stubFetch(
      jsonResponse(200, {
        inmobiliarias: [
          { id: 'inm-1', razonSocial: 'Lotes del Sur', cuit: '30712345678' },
          { id: 'inm-2', razonSocial: 'Altamira', cuit: null, telefono: null, email: null },
        ],
      }),
    )

    await expect(listAgencies('token-123')).resolves.toEqual([
      {
        id: 'inm-1',
        razonSocial: 'Lotes del Sur',
        cuit: '30712345678',
        telefono: '',
        email: '',
      },
      { id: 'inm-2', razonSocial: 'Altamira', cuit: '', telefono: '', email: '' },
    ])
  })

  it('sends the access token as a bearer header', async () => {
    const fetchMock = stubFetch(jsonResponse(200, { inmobiliarias: [] }))

    await listAgencies('token-123')

    const [, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer token-123')
  })

  it('treats a null list as no agencies', async () => {
    stubFetch(jsonResponse(200, { inmobiliarias: null }))

    await expect(listAgencies('token-123')).resolves.toEqual([])
  })

  it('rejects a payload without the inmobiliarias key', async () => {
    stubFetch(jsonResponse(200, { otros: [] }))

    await expect(listAgencies('token-123')).rejects.toThrow(GENERIC_ERROR)
  })

  it('rejects a list whose items are not agencies', async () => {
    stubFetch(jsonResponse(200, { inmobiliarias: [{ id: 'inm-1' }] }))

    await expect(listAgencies('token-123')).rejects.toThrow(GENERIC_ERROR)
  })

  // An optional field of the wrong type used to survive the type guard and
  // blow up later, when the search called .includes on a number.
  it('rejects an agency whose optional field is not a string', async () => {
    stubFetch(
      jsonResponse(200, {
        inmobiliarias: [{ id: 'inm-1', razonSocial: 'Lotes del Sur', cuit: 42 }],
      }),
    )

    await expect(listAgencies('token-123')).rejects.toThrow(GENERIC_ERROR)
  })

  it('surfaces the message of an error response', async () => {
    stubFetch(jsonResponse(403, { code: 'forbidden', message: 'No tenés permisos' }))

    await expect(listAgencies('token-123')).rejects.toThrow('No tenés permisos')
  })

  it('reports a request that never reached the server as a network failure', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => {
        throw new TypeError('Failed to fetch')
      }),
    )

    await expect(listAgencies('token-123')).rejects.toMatchObject({ code: 'network_error' })
  })

  it('keeps an aborted request as an AbortError', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => {
        throw new DOMException('The operation was aborted.', 'AbortError')
      }),
    )

    await expect(listAgencies('token-123')).rejects.toMatchObject({ name: 'AbortError' })
  })

  it('rejects a successful response that is not JSON', async () => {
    stubFetch(new Response('not json', { status: 200 }))

    await expect(listAgencies('token-123')).rejects.toThrow(GENERIC_ERROR)
  })
})

describe('createAgency', () => {
  it('posts the form values and returns the created agency', async () => {
    const fetchMock = stubFetch(
      jsonResponse(201, { id: 'inm-1', razonSocial: 'Lotes del Sur', cuit: '30712345678' }),
    )

    const created = await createAgency('token-123', {
      razonSocial: 'Lotes del Sur',
      cuit: '30712345678',
      telefono: '',
      email: '',
    })

    expect(created.id).toBe('inm-1')
    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toContain('/api/v1/inmobiliarias')
    expect(init.method).toBe('POST')
    expect(JSON.parse(String(init.body))).toEqual({
      razonSocial: 'Lotes del Sur',
      cuit: '30712345678',
      telefono: '',
      email: '',
    })
  })

  it('rejects a response that is not an agency', async () => {
    stubFetch(jsonResponse(201, { id: 'inm-1' }))

    await expect(
      createAgency('token-123', {
        razonSocial: 'Lotes del Sur',
        cuit: '',
        telefono: '',
        email: '',
      }),
    ).rejects.toThrow(GENERIC_ERROR)
  })

  it('surfaces a conflict with the backend code', async () => {
    stubFetch(jsonResponse(409, { code: 'cuit_in_use', message: 'El CUIT ya está en uso' }))

    await expect(
      createAgency('token-123', {
        razonSocial: 'Lotes del Sur',
        cuit: '30712345678',
        telefono: '',
        email: '',
      }),
    ).rejects.toThrow(ApiError)
  })
})

describe('updateAgency', () => {
  it('patches the agency by id', async () => {
    const fetchMock = stubFetch(
      jsonResponse(200, { id: 'inm-1', razonSocial: 'Lotes del Sur SRL' }),
    )

    const updated = await updateAgency('token-123', 'inm-1', {
      razonSocial: 'Lotes del Sur SRL',
      cuit: '',
      telefono: '',
      email: '',
    })

    expect(updated.razonSocial).toBe('Lotes del Sur SRL')
    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toContain('/api/v1/inmobiliarias/inm-1')
    expect(init.method).toBe('PATCH')
  })
})

describe('deleteAgency', () => {
  it('deletes the agency by id', async () => {
    const fetchMock = stubFetch(new Response(null, { status: 204 }))

    await deleteAgency('token-123', 'inm-1')

    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toContain('/api/v1/inmobiliarias/inm-1')
    expect(init.method).toBe('DELETE')
  })

  it('surfaces the message of an error response', async () => {
    stubFetch(
      jsonResponse(404, { code: 'agency_not_found', message: 'Inmobiliaria no encontrada' }),
    )

    await expect(deleteAgency('token-123', 'inm-1')).rejects.toThrow(
      'Inmobiliaria no encontrada',
    )
  })
})
