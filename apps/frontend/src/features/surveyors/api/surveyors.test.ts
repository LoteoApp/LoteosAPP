import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  createSurveyor,
  deactivateSurveyor,
  listSurveyors,
  updateSurveyor,
} from './surveyors'

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  } as unknown as Response
}

function stubFetch(response: Response) {
  const fetchMock = vi.fn().mockResolvedValue(response)
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

const surveyor = {
  id: 'agri-1',
  nombre: 'Ana',
  apellido: 'Gómez',
  email: 'ana@example.com',
  fechaBaja: null,
}

beforeEach(() => {
  vi.unstubAllGlobals()
})

describe('listSurveyors', () => {
  it('requests the active surveyors with the bearer token', async () => {
    const fetchMock = stubFetch(jsonResponse({ agrimensores: [surveyor] }))

    await expect(listSurveyors('token-123', false)).resolves.toEqual([surveyor])

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toContain('/api/v1/agrimensores')
    expect(url).not.toContain('incluirBajas')
    expect(init.headers.Authorization).toBe('Bearer token-123')
  })

  it('asks for the inactive surveyors too when requested', async () => {
    const fetchMock = stubFetch(jsonResponse({ agrimensores: [] }))

    await listSurveyors('token-123', true)

    expect(fetchMock.mock.calls[0][0]).toContain('incluirBajas=true')
  })

  it('returns an empty list when the body carries no surveyors', async () => {
    stubFetch(jsonResponse({}))

    await expect(listSurveyors('token-123', false)).resolves.toEqual([])
  })

  it('throws the message the API returned', async () => {
    stubFetch(jsonResponse({ code: 'forbidden', message: 'No tenés permisos' }, 403))

    await expect(listSurveyors('token-123', false)).rejects.toThrow('No tenés permisos')
  })

  it('throws a generic message when the error body is not readable', async () => {
    stubFetch({
      ok: false,
      status: 500,
      json: async () => {
        throw new Error('not json')
      },
    } as unknown as Response)

    await expect(listSurveyors('token-123', false)).rejects.toThrow(
      'No se pudo completar la operación, intentá nuevamente.',
    )
  })

  it('throws a generic message when the error body has no message', async () => {
    stubFetch(jsonResponse({ code: 'internal_error' }, 500))

    await expect(listSurveyors('token-123', false)).rejects.toThrow(
      'No se pudo completar la operación, intentá nuevamente.',
    )
  })
})

describe('createSurveyor', () => {
  it('posts the form values and returns the temporary password', async () => {
    const fetchMock = stubFetch(
      jsonResponse({ ...surveyor, temporaryPassword: 'temp-pass-123' }, 201),
    )

    const created = await createSurveyor('token-123', {
      nombre: 'Ana',
      apellido: 'Gómez',
      email: 'ana@example.com',
    })

    expect(created.temporaryPassword).toBe('temp-pass-123')
    expect(created.surveyor.id).toBe('agri-1')

    const [, init] = fetchMock.mock.calls[0]
    expect(init.method).toBe('POST')
    expect(init.headers['Content-Type']).toBe('application/json')
    expect(JSON.parse(init.body)).toEqual({
      nombre: 'Ana',
      apellido: 'Gómez',
      email: 'ana@example.com',
    })
  })

  it('falls back to an empty temporary password when the API omits it', async () => {
    stubFetch(jsonResponse(surveyor, 201))

    const created = await createSurveyor('token-123', {
      nombre: 'Ana',
      apellido: 'Gómez',
      email: 'ana@example.com',
    })

    expect(created.temporaryPassword).toBe('')
  })
})

describe('updateSurveyor', () => {
  it('patches only nombre and apellido', async () => {
    const fetchMock = stubFetch(jsonResponse({ ...surveyor, nombre: 'Ana María' }))

    await expect(
      updateSurveyor('token-123', 'agri-1', { nombre: 'Ana María', apellido: 'Gómez' }),
    ).resolves.toMatchObject({ nombre: 'Ana María' })

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toContain('/api/v1/agrimensores/agri-1')
    expect(init.method).toBe('PATCH')
    expect(JSON.parse(init.body)).toEqual({ nombre: 'Ana María', apellido: 'Gómez' })
  })
})

describe('deactivateSurveyor', () => {
  it('sends a DELETE for the given id', async () => {
    const fetchMock = stubFetch({
      ok: true,
      status: 204,
      json: async () => ({}),
    } as unknown as Response)

    await expect(deactivateSurveyor('token-123', 'agri-1')).resolves.toBeUndefined()

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toContain('/api/v1/agrimensores/agri-1')
    expect(init.method).toBe('DELETE')
  })

  it('throws the message the API returned', async () => {
    stubFetch(jsonResponse({ message: 'El agrimensor ya está dado de baja' }, 409))

    await expect(deactivateSurveyor('token-123', 'agri-1')).rejects.toThrow(
      'El agrimensor ya está dado de baja',
    )
  })
})
