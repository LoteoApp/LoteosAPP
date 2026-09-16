import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError, apiFetch, apiFetchBlob, messageFromError } from './client'
import { apiUrl } from '../config/env'
import { supabaseClient } from '../config/supabase-client'

vi.mock('../config/supabase-client', () => ({
  supabaseClient: { auth: { signOut: vi.fn() } },
}))

const signOutMock = vi.mocked(supabaseClient.auth.signOut)

afterEach(() => {
  vi.unstubAllGlobals()
  signOutMock.mockReset()
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

  it('forwards the abort signal to fetch', async () => {
    const mock = stubFetch(new Response('{}', { status: 200 }))
    const controller = new AbortController()

    await apiFetch('/api/v1/loteos', { signal: controller.signal })

    const [, init] = mock.mock.calls[0]
    expect(init?.signal).toBe(controller.signal)
  })

  it('forwards an abort signal and rethrows its AbortError untouched', async () => {
    const mock = stubFetch(() => Promise.reject(new DOMException('aborted', 'AbortError')))
    const controller = new AbortController()

    const error = await apiFetch('/api/v1/loteos', { signal: controller.signal }).catch(
      (caught: unknown) => caught,
    )

    expect(error).toBeInstanceOf(DOMException)
    expect((error as DOMException).name).toBe('AbortError')
    expect(mock.mock.calls[0][1]?.signal).toBe(controller.signal)
  })

  it('signs out and sends the browser to /login on a 401, without resolving or rejecting', async () => {
    stubFetch(
      new Response(JSON.stringify({ code: 'unauthorized', message: 'No autorizado' }), {
        status: 401,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    signOutMock.mockResolvedValue({ error: null })
    const location = { href: '' }
    vi.stubGlobal('location', location)

    let settled = false
    apiFetch('/api/v1/usuarios').then(
      () => (settled = true),
      () => (settled = true),
    )
    await vi.waitFor(() => expect(location.href).toBe('/login'))

    expect(signOutMock).toHaveBeenCalledOnce()
    expect(settled).toBe(false)
  })

  it('signs out and sends the browser to /login when the account is given de baja', async () => {
    stubFetch(
      new Response(JSON.stringify({ code: 'account_inactive', message: 'Tu cuenta fue dada de baja' }), {
        status: 403,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    signOutMock.mockResolvedValue({ error: null })
    const location = { href: '' }
    vi.stubGlobal('location', location)

    void apiFetch('/api/v1/usuarios')
    await vi.waitFor(() => expect(location.href).toBe('/login'))
  })

  it('redirects to /login even when signing out fails', async () => {
    stubFetch(
      new Response(JSON.stringify({ code: 'unauthorized', message: 'No autorizado' }), {
        status: 401,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    signOutMock.mockRejectedValue(new Error('network down'))
    const location = { href: '' }
    vi.stubGlobal('location', location)

    void apiFetch('/api/v1/usuarios')
    await vi.waitFor(() => expect(location.href).toBe('/login'))
  })

  it('redirects to /login even when signing out never resolves', async () => {
    vi.useFakeTimers()
    try {
      stubFetch(
        new Response(JSON.stringify({ code: 'unauthorized', message: 'No autorizado' }), {
          status: 401,
          headers: { 'Content-Type': 'application/json' },
        }),
      )
      signOutMock.mockReturnValue(new Promise(() => {}))
      const location = { href: '' }
      vi.stubGlobal('location', location)

      void apiFetch('/api/v1/usuarios')
      await vi.advanceTimersByTimeAsync(3000)

      expect(location.href).toBe('/login')
    } finally {
      vi.useRealTimers()
    }
  })

  it('does not redirect for a 403 unrelated to the account being inactive', async () => {
    stubFetch(
      new Response(JSON.stringify({ code: 'forbidden', message: 'No tenés permisos' }), {
        status: 403,
        headers: { 'Content-Type': 'application/json' },
      }),
    )

    await expect(apiFetch('/api/v1/usuarios')).rejects.toMatchObject({ code: 'forbidden' })
    expect(signOutMock).not.toHaveBeenCalled()
  })
})

describe('apiFetchBlob', () => {
  it('returns a Blob and sends the bearer token through the shared request path', async () => {
    const mock = stubFetch(
      new Response('%PDF-1.4', {
        status: 200,
        headers: { 'Content-Type': 'application/pdf' },
      }),
    )

    const result = await apiFetchBlob('/api/v1/reservas/1/comprobante', { token: 'tok' })

    expect(result.type).toBe('application/pdf')
    expect(await result.text()).toBe('%PDF-1.4')
    const [, init] = mock.mock.calls[0]
    expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer tok')
  })

  it('preserves the backend error code, message and status', async () => {
    stubFetch(
      new Response(
        JSON.stringify({ code: 'reservation_not_found', message: 'La reserva solicitada no existe' }),
        { status: 404, headers: { 'Content-Type': 'application/json' } },
      ),
    )

    await expect(apiFetchBlob('/api/v1/reservas/missing/comprobante')).rejects.toMatchObject({
      code: 'reservation_not_found',
      message: 'La reserva solicitada no existe',
      status: 404,
    })
  })

  it.each([
    [401, 'unauthorized'],
    [403, 'account_inactive'],
  ])('signs out and redirects for session-blocking status %s', async (status, code) => {
    stubFetch(
      new Response(JSON.stringify({ code, message: 'La sesión no puede continuar' }), {
        status,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    signOutMock.mockResolvedValue({ error: null })
    const location = { href: '' }
    vi.stubGlobal('location', location)

    let settled = false
    void apiFetchBlob('/api/v1/reservas/1/comprobante').then(
      () => (settled = true),
      () => (settled = true),
    )
    await vi.waitFor(() => expect(location.href).toBe('/login'))

    expect(signOutMock).toHaveBeenCalledOnce()
    expect(settled).toBe(false)
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
