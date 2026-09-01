import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Session, User } from '@supabase/supabase-js'
import ClientsPage from './ClientsPage'
import { AuthContext, type AuthContextValue } from '../../auth/hooks/use-auth'

type StoredCliente = {
  id: string
  nombre: string
  apellido: string
  dni: string
  celular?: string
  email?: string
}

let stored: StoredCliente[] = []
let failure: { status: number; message: string } | null = null
let nextId = 0

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

// A minimal stand-in for /api/v1/clientes that keeps the created rows, so a
// test can assert what the screen shows after it reloads from the API rather
// than what it kept in local state.
function installFetch() {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      const method = init?.method ?? 'GET'

      if (failure) {
        return jsonResponse(failure.status, { code: 'error', message: failure.message })
      }

      if (method === 'GET') {
        return jsonResponse(200, { clientes: stored })
      }

      if (method === 'POST') {
        const values = JSON.parse(String(init?.body)) as Omit<StoredCliente, 'id'>
        if (stored.some((cliente) => cliente.dni === values.dni)) {
          return jsonResponse(409, { code: 'dni_in_use', message: 'El DNI ya está en uso' })
        }
        nextId += 1
        const created = { id: `cliente-${nextId}`, ...values }
        stored = [...stored, created]
        return jsonResponse(201, created)
      }

      const id = url.slice(url.lastIndexOf('/') + 1)

      if (method === 'PATCH') {
        const values = JSON.parse(String(init?.body)) as Partial<StoredCliente>
        stored = stored.map((cliente) =>
          cliente.id === id ? { ...cliente, ...values } : cliente
        )
        return jsonResponse(200, stored.find((cliente) => cliente.id === id))
      }

      if (method === 'DELETE') {
        stored = stored.filter((cliente) => cliente.id !== id)
        return new Response(null, { status: 204 })
      }

      return jsonResponse(405, { code: 'method_not_allowed', message: 'Método no permitido' })
    })
  )
}

function renderClientsPage(role: string | null = 'administrador') {
  const value: AuthContextValue = {
    isLoading: false,
    session: { access_token: 'token-123' } as unknown as Session,
    user: role ? ({ app_metadata: { role } } as unknown as User) : null,
    error: null,
    login: vi.fn(),
    logout: vi.fn(),
  }

  render(
    <AuthContext.Provider value={value}>
      <ClientsPage />
    </AuthContext.Provider>,
  )
}

async function fillClientForm(
  user: ReturnType<typeof userEvent.setup>,
  values: { nombre: string; apellido: string; dni: string; celular?: string; email?: string }
) {
  await user.type(screen.getByLabelText('Nombre'), values.nombre)
  await user.type(screen.getByLabelText('Apellido'), values.apellido)
  await user.type(screen.getByLabelText('DNI'), values.dni)
  if (values.celular) {
    await user.type(screen.getByLabelText('Celular'), values.celular)
  }
  if (values.email) {
    await user.type(screen.getByLabelText('Correo electrónico'), values.email)
  }
}

beforeEach(() => {
  stored = []
  failure = null
  nextId = 0
  installFetch()
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('ClientsPage', () => {
  it('renders the section heading', () => {
    renderClientsPage()

    expect(screen.getByRole('heading', { name: 'Clientes' })).toBeInTheDocument()
  })

  it('lists the clients already stored in the backend', async () => {
    stored = [
      { id: 'cliente-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' },
      { id: 'cliente-2', nombre: 'Luis', apellido: 'Gómez', dni: '28555999' },
    ]
    renderClientsPage()

    expect(await screen.findByText('Ana Pérez')).toBeInTheDocument()
    expect(screen.getByText('Luis Gómez')).toBeInTheDocument()
  })

  it('shows an empty state when there are no clients', async () => {
    renderClientsPage()

    expect(await screen.findByText('No hay clientes cargados todavía.')).toBeInTheDocument()
  })

  it('shows the error returned by the backend when the list cannot be loaded', async () => {
    failure = { status: 403, message: 'Tu usuario no está habilitado para operar en el sistema' }
    renderClientsPage()

    expect(
      await screen.findByText('Tu usuario no está habilitado para operar en el sistema')
    ).toBeInTheDocument()
  })

  it('creates a new client and shows it in the list', async () => {
    const user = userEvent.setup()
    renderClientsPage()
    await screen.findByText('No hay clientes cargados todavía.')

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))
    await fillClientForm(user, { nombre: 'Ana', apellido: 'Pérez', dni: '30111222' })
    await user.click(screen.getByRole('button', { name: 'Crear cliente' }))

    expect(await screen.findByText('Ana Pérez')).toBeInTheDocument()
    expect(screen.getByText('DNI 30111222')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Crear cliente' })).not.toBeInTheDocument()
  })

  it('shows contact info when celular or email are provided', async () => {
    const user = userEvent.setup()
    renderClientsPage()
    await screen.findByText('No hay clientes cargados todavía.')

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))
    await fillClientForm(user, {
      nombre: 'Ana',
      apellido: 'Pérez',
      dni: '30111222',
      celular: '1122334455',
      email: 'ana@example.com',
    })
    await user.click(screen.getByRole('button', { name: 'Crear cliente' }))

    expect(await screen.findByText('1122334455 · ana@example.com')).toBeInTheDocument()
  })

  it('requires nombre, apellido and dni', async () => {
    const user = userEvent.setup()
    renderClientsPage()
    await screen.findByText('No hay clientes cargados todavía.')

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))
    await user.click(screen.getByRole('button', { name: 'Crear cliente' }))

    expect(await screen.findByText('Completá nombre, apellido y DNI.')).toBeInTheDocument()
  })

  it('shows the backend error when the alta is rejected', async () => {
    const user = userEvent.setup()
    renderClientsPage()
    await screen.findByText('No hay clientes cargados todavía.')

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))
    await fillClientForm(user, { nombre: 'Ana', apellido: 'Pérez', dni: '30111222' })
    failure = { status: 409, message: 'El DNI ya está en uso' }
    await user.click(screen.getByRole('button', { name: 'Crear cliente' }))

    expect(await screen.findByText('El DNI ya está en uso')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Crear cliente' })).toBeInTheDocument()
  })

  it('hides the empty state message while the form is open', async () => {
    const user = userEvent.setup()
    renderClientsPage()

    expect(await screen.findByText('No hay clientes cargados todavía.')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))

    expect(screen.queryByText('No hay clientes cargados todavía.')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Cancelar' }))

    expect(screen.getByText('No hay clientes cargados todavía.')).toBeInTheDocument()
  })

  it('rejects a duplicate dni before reaching the backend', async () => {
    const user = userEvent.setup()
    stored = [{ id: 'cliente-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' }]
    renderClientsPage()
    await screen.findByText('Ana Pérez')

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))
    await fillClientForm(user, { nombre: 'Luis', apellido: 'Gómez', dni: '30111222' })
    await user.click(screen.getByRole('button', { name: 'Crear cliente' }))

    expect(await screen.findByText('Ya existe un cliente con ese DNI.')).toBeInTheDocument()
    expect(screen.queryByText('Luis Gómez')).not.toBeInTheDocument()
  })

  it('edits an existing client without flagging its own dni as duplicate', async () => {
    const user = userEvent.setup()
    stored = [{ id: 'cliente-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' }]
    renderClientsPage()
    await screen.findByText('Ana Pérez')

    await user.click(screen.getByRole('button', { name: 'Editar' }))
    const nombreInput = screen.getByLabelText('Nombre')
    await user.clear(nombreInput)
    await user.type(nombreInput, 'Ana María')
    await user.click(screen.getByRole('button', { name: 'Guardar cambios' }))

    expect(await screen.findByText('Ana María Pérez')).toBeInTheDocument()
  })

  it('removes a client after confirming the baja inline', async () => {
    const user = userEvent.setup()
    stored = [{ id: 'cliente-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' }]
    renderClientsPage()
    await screen.findByText('Ana Pérez')

    await user.click(screen.getByRole('button', { name: 'Dar de baja' }))
    expect(screen.getByText('¿Confirmar baja?')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Confirmar' }))

    expect(await screen.findByText('No hay clientes cargados todavía.')).toBeInTheDocument()
    expect(screen.queryByText('Ana Pérez')).not.toBeInTheDocument()
  })

  it('keeps the client when the inline baja confirmation is cancelled', async () => {
    const user = userEvent.setup()
    stored = [{ id: 'cliente-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' }]
    renderClientsPage()
    await screen.findByText('Ana Pérez')

    await user.click(screen.getByRole('button', { name: 'Dar de baja' }))
    await user.click(screen.getByRole('button', { name: 'Cancelar' }))

    expect(screen.getByText('Ana Pérez')).toBeInTheDocument()
    expect(screen.queryByText('¿Confirmar baja?')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Dar de baja' })).toBeInTheDocument()
  })

  it('closes the form without saving when cancel is clicked', async () => {
    const user = userEvent.setup()
    renderClientsPage()
    await screen.findByText('No hay clientes cargados todavía.')

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))
    await fillClientForm(user, { nombre: 'Ana', apellido: 'Pérez', dni: '30111222' })
    await user.click(screen.getByRole('button', { name: 'Cancelar' }))

    expect(screen.queryByText('Ana Pérez')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Nuevo cliente' })).toBeInTheDocument()
  })

  it('filters the list by name, surname or dni', async () => {
    const user = userEvent.setup()
    stored = [
      { id: 'cliente-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' },
      { id: 'cliente-2', nombre: 'Luis', apellido: 'Gómez', dni: '28555999' },
    ]
    renderClientsPage()
    await screen.findByText('Ana Pérez')

    await user.type(screen.getByLabelText('Buscar'), 'gomez')

    expect(screen.queryByText('Ana Pérez')).not.toBeInTheDocument()
    expect(screen.getByText('Luis Gómez')).toBeInTheDocument()

    await user.clear(screen.getByLabelText('Buscar'))
    await user.type(screen.getByLabelText('Buscar'), '30111222')

    expect(screen.getByText('Ana Pérez')).toBeInTheDocument()
    expect(screen.queryByText('Luis Gómez')).not.toBeInTheDocument()
  })

  it('matches regardless of accents in the search term', async () => {
    const user = userEvent.setup()
    stored = [{ id: 'cliente-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' }]
    renderClientsPage()
    await screen.findByText('Ana Pérez')

    await user.type(screen.getByLabelText('Buscar'), 'Pérez')

    expect(screen.getByText('Ana Pérez')).toBeInTheDocument()
  })

  it('shows a message when the search has no matches', async () => {
    const user = userEvent.setup()
    stored = [{ id: 'cliente-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' }]
    renderClientsPage()
    await screen.findByText('Ana Pérez')

    await user.type(screen.getByLabelText('Buscar'), 'inexistente')

    expect(screen.getByText('No se encontraron clientes con esa búsqueda.')).toBeInTheDocument()
    expect(screen.queryByText('Ana Pérez')).not.toBeInTheDocument()
  })

  it('hides the list and the search box while a form is open', async () => {
    const user = userEvent.setup()
    stored = [{ id: 'cliente-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' }]
    renderClientsPage()
    await screen.findByText('Ana Pérez')

    expect(screen.getByLabelText('Buscar')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))

    expect(screen.queryByText('Ana Pérez')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Buscar')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Cancelar' }))

    expect(screen.getByText('Ana Pérez')).toBeInTheDocument()
    expect(screen.getByLabelText('Buscar')).toBeInTheDocument()
  })

  it('hides the list while editing a client', async () => {
    const user = userEvent.setup()
    stored = [{ id: 'cliente-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' }]
    renderClientsPage()
    await screen.findByText('Ana Pérez')

    await user.click(screen.getByRole('button', { name: 'Editar' }))

    expect(screen.queryByText('DNI 30111222')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Buscar')).not.toBeInTheDocument()
  })

  it('does not show the search box when there are no clients', async () => {
    renderClientsPage()
    await screen.findByText('No hay clientes cargados todavía.')

    expect(screen.queryByLabelText('Buscar')).not.toBeInTheDocument()
  })

  it('hides the baja action for a non-administrador role', async () => {
    stored = [{ id: 'cliente-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' }]
    renderClientsPage('administrativo')

    expect(await screen.findByText('Ana Pérez')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Dar de baja' })).not.toBeInTheDocument()
  })
})
