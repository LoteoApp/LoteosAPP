import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { User } from '@supabase/supabase-js'
import ClientsPage from './ClientsPage'
import { AuthContext, type AuthContextValue } from '../../auth/hooks/use-auth'

function renderClientsPage(role: string | null = 'administrador') {
  const value: AuthContextValue = {
    isLoading: false,
    session: null,
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

afterEach(() => {
  vi.restoreAllMocks()
})

describe('ClientsPage', () => {
  it('renders the section heading', () => {
    renderClientsPage()

    expect(screen.getByRole('heading', { name: 'Clientes' })).toBeInTheDocument()
  })

  it('shows an empty state when there are no clients', () => {
    renderClientsPage()

    expect(screen.getByText('No hay clientes cargados todavía.')).toBeInTheDocument()
  })

  it('creates a new client and shows it in the list', async () => {
    const user = userEvent.setup()
    renderClientsPage()

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))
    await fillClientForm(user, { nombre: 'Ana', apellido: 'Pérez', dni: '30111222' })
    await user.click(screen.getByRole('button', { name: 'Crear cliente' }))

    expect(screen.getByText('Ana Pérez')).toBeInTheDocument()
    expect(screen.getByText('DNI 30111222')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Crear cliente' })).not.toBeInTheDocument()
  })

  it('shows contact info when celular or email are provided', async () => {
    const user = userEvent.setup()
    renderClientsPage()

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))
    await fillClientForm(user, {
      nombre: 'Ana',
      apellido: 'Pérez',
      dni: '30111222',
      celular: '1122334455',
      email: 'ana@example.com',
    })
    await user.click(screen.getByRole('button', { name: 'Crear cliente' }))

    expect(screen.getByText('1122334455 · ana@example.com')).toBeInTheDocument()
  })

  it('requires nombre, apellido and dni', async () => {
    const user = userEvent.setup()
    renderClientsPage()

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))
    await user.click(screen.getByRole('button', { name: 'Crear cliente' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Completá nombre, apellido y DNI.')
  })

  it('hides the empty state message while the form is open', async () => {
    const user = userEvent.setup()
    renderClientsPage()

    expect(screen.getByText('No hay clientes cargados todavía.')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))

    expect(screen.queryByText('No hay clientes cargados todavía.')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Cancelar' }))

    expect(screen.getByText('No hay clientes cargados todavía.')).toBeInTheDocument()
  })

  it('rejects a duplicate dni', async () => {
    const user = userEvent.setup()
    renderClientsPage()

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))
    await fillClientForm(user, { nombre: 'Ana', apellido: 'Pérez', dni: '30111222' })
    await user.click(screen.getByRole('button', { name: 'Crear cliente' }))

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))
    await fillClientForm(user, { nombre: 'Luis', apellido: 'Gómez', dni: '30111222' })
    await user.click(screen.getByRole('button', { name: 'Crear cliente' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Ya existe un cliente con ese DNI.')
    expect(screen.queryByText('Luis Gómez')).not.toBeInTheDocument()
  })

  it('edits an existing client without flagging its own dni as duplicate', async () => {
    const user = userEvent.setup()
    renderClientsPage()

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))
    await fillClientForm(user, { nombre: 'Ana', apellido: 'Pérez', dni: '30111222' })
    await user.click(screen.getByRole('button', { name: 'Crear cliente' }))

    await user.click(screen.getByRole('button', { name: 'Editar' }))
    const nombreInput = screen.getByLabelText('Nombre')
    await user.clear(nombreInput)
    await user.type(nombreInput, 'Ana María')
    await user.click(screen.getByRole('button', { name: 'Guardar cambios' }))

    expect(screen.getByText('Ana María Pérez')).toBeInTheDocument()
  })

  it('removes a client after confirming the baja inline', async () => {
    const user = userEvent.setup()
    renderClientsPage()

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))
    await fillClientForm(user, { nombre: 'Ana', apellido: 'Pérez', dni: '30111222' })
    await user.click(screen.getByRole('button', { name: 'Crear cliente' }))

    await user.click(screen.getByRole('button', { name: 'Dar de baja' }))
    expect(screen.getByText('¿Confirmar baja?')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Confirmar' }))

    expect(screen.queryByText('Ana Pérez')).not.toBeInTheDocument()
    expect(screen.getByText('No hay clientes cargados todavía.')).toBeInTheDocument()
  })

  it('keeps the client when the inline baja confirmation is cancelled', async () => {
    const user = userEvent.setup()
    renderClientsPage()

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))
    await fillClientForm(user, { nombre: 'Ana', apellido: 'Pérez', dni: '30111222' })
    await user.click(screen.getByRole('button', { name: 'Crear cliente' }))

    await user.click(screen.getByRole('button', { name: 'Dar de baja' }))
    await user.click(screen.getByRole('button', { name: 'Cancelar' }))

    expect(screen.getByText('Ana Pérez')).toBeInTheDocument()
    expect(screen.queryByText('¿Confirmar baja?')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Dar de baja' })).toBeInTheDocument()
  })

  it('closes the form without saving when cancel is clicked', async () => {
    const user = userEvent.setup()
    renderClientsPage()

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))
    await fillClientForm(user, { nombre: 'Ana', apellido: 'Pérez', dni: '30111222' })
    await user.click(screen.getByRole('button', { name: 'Cancelar' }))

    expect(screen.queryByText('Ana Pérez')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Nuevo cliente' })).toBeInTheDocument()
  })

  it('filters the list by name, surname or dni', async () => {
    const user = userEvent.setup()
    renderClientsPage()

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))
    await fillClientForm(user, { nombre: 'Ana', apellido: 'Pérez', dni: '30111222' })
    await user.click(screen.getByRole('button', { name: 'Crear cliente' }))

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))
    await fillClientForm(user, { nombre: 'Luis', apellido: 'Gómez', dni: '28555999' })
    await user.click(screen.getByRole('button', { name: 'Crear cliente' }))

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
    renderClientsPage()

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))
    await fillClientForm(user, { nombre: 'Ana', apellido: 'Pérez', dni: '30111222' })
    await user.click(screen.getByRole('button', { name: 'Crear cliente' }))

    await user.type(screen.getByLabelText('Buscar'), 'Pérez')

    expect(screen.getByText('Ana Pérez')).toBeInTheDocument()
  })

  it('shows a message when the search has no matches', async () => {
    const user = userEvent.setup()
    renderClientsPage()

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))
    await fillClientForm(user, { nombre: 'Ana', apellido: 'Pérez', dni: '30111222' })
    await user.click(screen.getByRole('button', { name: 'Crear cliente' }))

    await user.type(screen.getByLabelText('Buscar'), 'inexistente')

    expect(screen.getByText('No se encontraron clientes con esa búsqueda.')).toBeInTheDocument()
    expect(screen.queryByText('Ana Pérez')).not.toBeInTheDocument()
  })

  it('hides the list and the search box while a form is open', async () => {
    const user = userEvent.setup()
    renderClientsPage()

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))
    await fillClientForm(user, { nombre: 'Ana', apellido: 'Pérez', dni: '30111222' })
    await user.click(screen.getByRole('button', { name: 'Crear cliente' }))

    expect(screen.getByText('Ana Pérez')).toBeInTheDocument()
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
    renderClientsPage()

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))
    await fillClientForm(user, { nombre: 'Ana', apellido: 'Pérez', dni: '30111222' })
    await user.click(screen.getByRole('button', { name: 'Crear cliente' }))

    await user.click(screen.getByRole('button', { name: 'Editar' }))

    expect(screen.queryByText('DNI 30111222')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Buscar')).not.toBeInTheDocument()
  })

  it('does not show the search box when there are no clients', () => {
    renderClientsPage()

    expect(screen.queryByLabelText('Buscar')).not.toBeInTheDocument()
  })

  it('hides the baja action for a non-administrador role', async () => {
    const user = userEvent.setup()
    renderClientsPage('administrativo')

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))
    await fillClientForm(user, { nombre: 'Ana', apellido: 'Pérez', dni: '30111222' })
    await user.click(screen.getByRole('button', { name: 'Crear cliente' }))

    expect(screen.getByText('Ana Pérez')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Dar de baja' })).not.toBeInTheDocument()
  })
})
