import { afterEach, describe, expect, it, vi } from 'vitest'
import { createUser, deactivateUser, listUsers, reactivateUser, updateUser } from './users'

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

describe('listUsers', () => {
  it('sends the bearer token, asks for inactive users too, and returns the users', async () => {
    const fetchMock = stubFetch(
      jsonResponse(200, {
        usuarios: [
          {
            id: 'usuario-1',
            email: 'ana@example.com',
            nombre: 'Ana',
            apellido: 'Pérez',
            rol: 'administrativo',
            perfilCompleto: true,
            fechaBaja: null,
            createdAt: '2026-01-01T00:00:00Z',
          },
        ],
      })
    )

    const usuarios = await listUsers('token-123')

    expect(usuarios).toHaveLength(1)
    expect(usuarios[0].email).toBe('ana@example.com')

    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toContain('/api/v1/usuarios')
    expect(url).toContain('incluirBajas=true')
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer token-123')
  })

  it('returns an empty list when usuarios is null', async () => {
    stubFetch(jsonResponse(200, { usuarios: null }))

    await expect(listUsers('token-123')).resolves.toEqual([])
  })

  it('rejects a 2xx body that does not carry the expected contract', async () => {
    stubFetch(jsonResponse(200, {}))

    await expect(listUsers('token-123')).rejects.toThrow(
      'No se pudo completar la operación, intentá nuevamente.'
    )
  })

  it('rejects a 2xx body whose usuarios are not users', async () => {
    stubFetch(jsonResponse(200, { usuarios: [{ id: 'usuario-1' }] }))

    await expect(listUsers('token-123')).rejects.toThrow(
      'No se pudo completar la operación, intentá nuevamente.'
    )
  })

  it('rejects a usuario with a rol outside GESTIONABLE_ROLES', async () => {
    stubFetch(
      jsonResponse(200, {
        usuarios: [
          {
            id: 'usuario-1',
            email: 'ana@example.com',
            nombre: 'Ana',
            apellido: 'Pérez',
            rol: 'administrador',
            perfilCompleto: true,
            fechaBaja: null,
            createdAt: '2026-01-01T00:00:00Z',
          },
        ],
      })
    )

    await expect(listUsers('token-123')).rejects.toThrow(
      'No se pudo completar la operación, intentá nuevamente.'
    )
  })

  it('rejects a usuario missing perfilCompleto, fechaBaja or createdAt', async () => {
    stubFetch(
      jsonResponse(200, {
        usuarios: [
          {
            id: 'usuario-1',
            email: 'ana@example.com',
            nombre: 'Ana',
            apellido: 'Pérez',
            rol: 'administrativo',
          },
        ],
      })
    )

    await expect(listUsers('token-123')).rejects.toThrow(
      'No se pudo completar la operación, intentá nuevamente.'
    )
  })

  it('forwards the abort signal to fetch', async () => {
    const fetchMock = stubFetch(jsonResponse(200, { usuarios: [] }))
    const controller = new AbortController()

    await listUsers('token-123', controller.signal)

    const [, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(init.signal).toBe(controller.signal)
  })

  it('throws the message returned by the backend', async () => {
    stubFetch(
      jsonResponse(403, {
        code: 'actor_not_provisioned',
        message: 'Tu usuario no está habilitado para operar en el sistema',
      })
    )

    await expect(listUsers('token-123')).rejects.toThrow(
      'Tu usuario no está habilitado para operar en el sistema'
    )
  })

  it('falls back to a generic message when the error body is not json', async () => {
    stubFetch(new Response('boom', { status: 500 }))

    await expect(listUsers('token-123')).rejects.toThrow(/error inesperado/i)
  })
})

describe('createUser', () => {
  it('posts the form values and returns the user with its temporary password', async () => {
    const fetchMock = stubFetch(
      jsonResponse(201, {
        id: 'usuario-1',
        email: 'ana@example.com',
        nombre: 'Ana',
        apellido: 'Pérez',
        rol: 'administrativo',
        perfilCompleto: true,
        fechaBaja: null,
        createdAt: '2026-01-01T00:00:00Z',
        temporaryPassword: 'temp-pass-123',
      })
    )

    const created = await createUser('token-123', {
      nombre: 'Ana',
      apellido: 'Pérez',
      email: 'ana@example.com',
      rol: 'administrativo',
    })

    expect(created.usuario.id).toBe('usuario-1')
    expect(created.temporaryPassword).toBe('temp-pass-123')

    const [, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(init.method).toBe('POST')
    expect((init.headers as Record<string, string>)['Content-Type']).toBe('application/json')
    expect(JSON.parse(String(init.body))).toEqual({
      nombre: 'Ana',
      apellido: 'Pérez',
      email: 'ana@example.com',
      rol: 'administrativo',
    })
  })

  it('rejects a created user with no temporary password', async () => {
    stubFetch(
      jsonResponse(201, {
        id: 'usuario-1',
        email: 'ana@example.com',
        nombre: 'Ana',
        apellido: 'Pérez',
        rol: 'administrativo',
        perfilCompleto: true,
        fechaBaja: null,
        createdAt: '2026-01-01T00:00:00Z',
      })
    )

    await expect(
      createUser('token-123', {
        nombre: 'Ana',
        apellido: 'Pérez',
        email: 'ana@example.com',
        rol: 'administrativo',
      })
    ).rejects.toThrow('No se pudo completar la operación, intentá nuevamente.')
  })

  it('throws the conflict message when the email is taken', async () => {
    stubFetch(jsonResponse(409, { code: 'email_in_use', message: 'El email ya está en uso' }))

    await expect(
      createUser('token-123', {
        nombre: 'Ana',
        apellido: 'Pérez',
        email: 'ana@example.com',
        rol: 'administrativo',
      })
    ).rejects.toThrow('El email ya está en uso')
  })
})

describe('updateUser', () => {
  it('patches the user by id', async () => {
    const fetchMock = stubFetch(
      jsonResponse(200, {
        id: 'usuario-1',
        email: 'ana@example.com',
        nombre: 'Ana María',
        apellido: 'Pérez',
        rol: 'administrativo',
        perfilCompleto: true,
        fechaBaja: null,
        createdAt: '2026-01-01T00:00:00Z',
      })
    )

    const usuario = await updateUser('token-123', 'usuario-1', {
      nombre: 'Ana María',
      apellido: 'Pérez',
    })

    expect(usuario.nombre).toBe('Ana María')

    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toContain('/api/v1/usuarios/usuario-1')
    expect(init.method).toBe('PATCH')
  })
})

describe('deactivateUser', () => {
  it('deletes the user by id', async () => {
    const fetchMock = stubFetch(new Response(null, { status: 204 }))

    await deactivateUser('token-123', 'usuario-1')

    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toContain('/api/v1/usuarios/usuario-1')
    expect(init.method).toBe('DELETE')
  })

  it('throws the message returned by the backend', async () => {
    stubFetch(
      jsonResponse(409, { code: 'user_already_inactive', message: 'El usuario ya está dado de baja' })
    )

    await expect(deactivateUser('token-123', 'usuario-1')).rejects.toThrow(
      'El usuario ya está dado de baja'
    )
  })
})

describe('reactivateUser', () => {
  it('posts to the reactivar endpoint and returns the reactivated user', async () => {
    const fetchMock = stubFetch(
      jsonResponse(200, {
        id: 'usuario-1',
        email: 'ana@example.com',
        nombre: 'Ana',
        apellido: 'Pérez',
        rol: 'administrativo',
        perfilCompleto: true,
        fechaBaja: null,
        createdAt: '2026-01-01T00:00:00Z',
      })
    )

    const usuario = await reactivateUser('token-123', 'usuario-1')

    expect(usuario.fechaBaja).toBeNull()

    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toContain('/api/v1/usuarios/usuario-1/reactivar')
    expect(init.method).toBe('POST')
  })

  it('throws the message returned by the backend', async () => {
    stubFetch(jsonResponse(409, { code: 'user_already_active', message: 'El usuario ya está activo' }))

    await expect(reactivateUser('token-123', 'usuario-1')).rejects.toThrow('El usuario ya está activo')
  })
})
