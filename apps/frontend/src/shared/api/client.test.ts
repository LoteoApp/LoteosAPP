import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError, apiFetch, messageFromError } from './client'
import { apiUrl } from '../config/env'

afterEach(() => {
  vi.unstubAllGlobals()
})

function stubFetch(response: Response | (() => Promise<Response>)) {
  const mock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) =>
    typeof response === 'function' ? response() : response,
  )
  vi.stubGlobal('fetch', mock)
  return mock
}

describe('apiFetch', () => {
  it('sends JSON with a bearer token and parses the response', async () => {
    const mock = stubFetch(
      new Response(JSON.stringify({ id: 'loteo-1' }), {
        status: 201,
        headers: { 'Content-Type': 'application/json' },
      }),
    )

    const result = await apiFetch<{ id: string }>('/api/v1/loteos', {
      method: 'POST',
      body: { nombre: 'x' },
      token: 'tok',
    })

    expect(result).toEqual({ id: 'loteo-1' })
    const [url, init] = mock.mock.calls[0]
    expect(url).toBe(`${apiUrl}/api/v1/loteos`)
    expect(init?.method).toBe('POST')
    expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer tok')
    expect(new Headers(init?.headers).get('Content-Type')).toBe('application/json')
    expect(init?.body).toBe('{"nombre":"x"}')
  })

  it('passes a FormData body through without forcing a content type', async () => {
    const mock = stubFetch(new Response(null, { status: 204 }))
    const form = new FormData()
    form.append('archivo', new File(['x'], 'a.dxf'))

    const result = await apiFetch('/api/v1/loteos/1/dxf', { method: 'PUT', body: form })

    expect(result).toBeUndefined()
    const [, init] = mock.mock.calls[0]
    expect(init?.body).toBe(form)
    expect(new Headers(init?.headers).has('Content-Type')).toBe(false)
  })

  it('omits the Authorization header when no token is given', async () => {
    const mock = stubFetch(new Response('{}', { status: 200 }))

    await apiFetch('/api/v1/loteos')

    const [, init] = mock.mock.calls[0]
    expect(new Headers(init?.headers).has('Authorization')).toBe(false)
  })

  it('throws an ApiError carrying the backend code and message', async () => {
    stubFetch(
      new Response(JSON.stringify({ code: 'invalid_loteo_nombre', message: 'Falta el nombre' }), {
        status: 400,
        headers: { 'Content-Type': 'application/json' },
      }),
    )

    await expect(apiFetch('/api/v1/loteos', { method: 'POST', body: {} })).rejects.toMatchObject({
      code: 'invalid_loteo_nombre',
      message: 'Falta el nombre',
      status: 400,
    })
  })

  it('falls back to a generic message for a non-JSON server error', async () => {
    stubFetch(new Response('<html>oops</html>', { status: 500 }))

    const error = await apiFetch('/api/v1/loteos').catch((caught: unknown) => caught)

    expect(error).toBeInstanceOf(ApiError)
    expect((error as ApiError).status).toBe(500)
    expect((error as ApiError).message).toMatch(/error inesperado/i)
  })

  it('wraps a network failure in an ApiError', async () => {
    stubFetch(() => Promise.reject(new TypeError('Failed to fetch')))

    const error = await apiFetch('/api/v1/loteos').catch((caught: unknown) => caught)

    expect(error).toBeInstanceOf(ApiError)
    expect((error as ApiError).code).toBe('network_error')
    expect((error as ApiError).status).toBe(0)
  })
})

describe('messageFromError', () => {
  it('returns the ApiError message', () => {
    expect(messageFromError(new ApiError('Sin permisos', 'forbidden', 403))).toBe('Sin permisos')
  })

  it('returns a generic message for anything else', () => {
    expect(messageFromError(new Error('boom'))).toMatch(/error inesperado/i)
  })
})
