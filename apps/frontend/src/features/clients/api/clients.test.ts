import { afterEach, describe, expect, it, vi } from 'vitest'
import { createClient, deleteClient, listClients, updateClient } from './clients'

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

describe('listClients', () => {
  it('sends the bearer token and returns the clients', async () => {
    const fetchMock = stubFetch(
      jsonResponse(200, {
        clientes: [
          {
            id: 'cliente-1',
            nombre: 'Ana',
            apellido: 'Pérez',
            dni: '30111222',
            celular: '1122334455',
            email: 'ana@example.com',
          },
        ],
      })
    )

    const clientes = await listClients('token-123')

    expect(clientes).toEqual([
      {
        id: 'cliente-1',
        nombre: 'Ana',
        apellido: 'Pérez',
        dni: '30111222',
        celular: '1122334455',
        email: 'ana@example.com',
      },
    ])

    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toContain('/api/v1/clientes')
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer token-123')
  })

  it('turns an absent celular or email into an empty string', async () => {
    stubFetch(
      jsonResponse(200, {
        clientes: [{ id: 'cliente-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' }],
      })
    )

    const [cliente] = await listClients('token-123')

    expect(cliente.celular).toBe('')
    expect(cliente.email).toBe('')
  })

  it('returns an empty list when clientes is null', async () => {
    stubFetch(jsonResponse(200, { clientes: null }))

    await expect(listClients('token-123')).resolves.toEqual([])
  })

  it('rejects a 2xx body that does not carry the expected contract', async () => {
    stubFetch(jsonResponse(200, {}))

    await expect(listClients('token-123')).rejects.toThrow(
      'No se pudo completar la operación, intentá nuevamente.'
    )
  })

  it('rejects a 2xx body whose clientes are not clients', async () => {
    stubFetch(jsonResponse(200, { clientes: [{ id: 'cliente-1' }] }))

    await expect(listClients('token-123')).rejects.toThrow(
      'No se pudo completar la operación, intentá nuevamente.'
    )
  })

  it('rejects a 2xx body that is not json', async () => {
    stubFetch(new Response('not json', { status: 200 }))

    await expect(listClients('token-123')).rejects.toThrow(
      'No se pudo completar la operación, intentá nuevamente.'
    )
  })

  it('forwards the abort signal to fetch', async () => {
    const fetchMock = stubFetch(jsonResponse(200, { clientes: [] }))
    const controller = new AbortController()

    await listClients('token-123', controller.signal)

    const [, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(init.signal).toBe(controller.signal)
  })

  it('throws the message returned by the backend', async () => {
    stubFetch(jsonResponse(403, { code: 'forbidden', message: 'No autorizado' }))

    await expect(listClients('token-123')).rejects.toThrow('No autorizado')
  })

  it('falls back to a generic message when the error body is not json', async () => {
    stubFetch(new Response('boom', { status: 500 }))

    await expect(listClients('token-123')).rejects.toThrow(
      'No se pudo completar la operación, intentá nuevamente.'
    )
  })
})

describe('createClient', () => {
  it('posts the form values', async () => {
    const fetchMock = stubFetch(
      jsonResponse(201, {
        id: 'cliente-1',
        nombre: 'Ana',
        apellido: 'Pérez',
        dni: '30111222',
      })
    )

    const cliente = await createClient('token-123', {
      nombre: 'Ana',
      apellido: 'Pérez',
      dni: '30111222',
      celular: '',
      email: '',
    })

    expect(cliente.id).toBe('cliente-1')

    const [, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(init.method).toBe('POST')
    expect((init.headers as Record<string, string>)['Content-Type']).toBe('application/json')
    expect(JSON.parse(String(init.body))).toMatchObject({ nombre: 'Ana', dni: '30111222' })
  })

  it('rejects a malformed created client', async () => {
    stubFetch(jsonResponse(201, { id: 'cliente-1' }))

    await expect(
      createClient('token-123', {
        nombre: 'Ana',
        apellido: 'Pérez',
        dni: '30111222',
        celular: '',
        email: '',
      })
    ).rejects.toThrow('No se pudo completar la operación, intentá nuevamente.')
  })

  it('throws the conflict message when the dni is taken', async () => {
    stubFetch(jsonResponse(409, { code: 'dni_in_use', message: 'El DNI ya está en uso' }))

    await expect(
      createClient('token-123', {
        nombre: 'Ana',
        apellido: 'Pérez',
        dni: '30111222',
        celular: '',
        email: '',
      })
    ).rejects.toThrow('El DNI ya está en uso')
  })
})

describe('updateClient', () => {
  it('patches the client by id', async () => {
    const fetchMock = stubFetch(
      jsonResponse(200, {
        id: 'cliente-1',
        nombre: 'Ana María',
        apellido: 'Pérez',
        dni: '30111222',
      })
    )

    const cliente = await updateClient('token-123', 'cliente-1', {
      nombre: 'Ana María',
      apellido: 'Pérez',
      dni: '30111222',
      celular: '',
      email: '',
    })

    expect(cliente.nombre).toBe('Ana María')

    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toContain('/api/v1/clientes/cliente-1')
    expect(init.method).toBe('PATCH')
  })
})

describe('deleteClient', () => {
  it('deletes the client by id', async () => {
    const fetchMock = stubFetch(new Response(null, { status: 204 }))

    await deleteClient('token-123', 'cliente-1')

    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toContain('/api/v1/clientes/cliente-1')
    expect(init.method).toBe('DELETE')
  })

  it('throws the message returned by the backend', async () => {
    stubFetch(jsonResponse(404, { code: 'client_not_found', message: 'Cliente no encontrado' }))

    await expect(deleteClient('token-123', 'cliente-1')).rejects.toThrow('Cliente no encontrado')
  })
})
