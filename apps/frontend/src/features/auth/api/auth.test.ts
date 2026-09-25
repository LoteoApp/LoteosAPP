import { afterEach, describe, expect, it, vi } from 'vitest'
import { confirmPasswordReset, requestPasswordReset } from './auth'

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

describe('requestPasswordReset', () => {
  it('posts the email to the recuperar-contrasena endpoint without a token', async () => {
    const fetchMock = stubFetch(new Response(null, { status: 204 }))

    await requestPasswordReset('ana@example.com')

    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toContain('/api/v1/auth/recuperar-contrasena')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body as string)).toEqual({ email: 'ana@example.com' })
    expect((init.headers as Record<string, string>).Authorization).toBeUndefined()
  })

  it('throws the message returned by the backend when the request is rate limited', async () => {
    stubFetch(
      jsonResponse(429, {
        code: 'password_reset_rate_limited',
        message: 'Ya te enviamos un mail. Esperá un momento antes de pedir otro',
      }),
    )

    await expect(requestPasswordReset('ana@example.com')).rejects.toThrow(
      'Ya te enviamos un mail. Esperá un momento antes de pedir otro',
    )
  })
})

describe('confirmPasswordReset', () => {
  it('posts the token and new password to the restablecer-contrasena endpoint', async () => {
    const fetchMock = stubFetch(new Response(null, { status: 204 }))

    await confirmPasswordReset('token-abc', 'a-new-password')

    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toContain('/api/v1/auth/restablecer-contrasena')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body as string)).toEqual({ token: 'token-abc', newPassword: 'a-new-password' })
  })

  it('throws the message returned by the backend for an invalid or expired token', async () => {
    stubFetch(
      jsonResponse(400, {
        code: 'password_reset_token_invalid',
        message: 'El link para restablecer la contraseña es inválido o venció',
      }),
    )

    await expect(confirmPasswordReset('token-abc', 'a-new-password')).rejects.toThrow(
      'El link para restablecer la contraseña es inválido o venció',
    )
  })
})
