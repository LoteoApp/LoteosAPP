import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import UsersPage from './UsersPage'
import { ROLE_LABELS, type GestionableRol } from '../types'

type StoredUsuario = {
  id: string
  email: string
  nombre: string
  apellido: string
  rol: string
  perfilCompleto: boolean
  fechaBaja: string | null
  createdAt: string
}

let stored: StoredUsuario[] = []
let failure: { status: number; message: string; code?: string } | null = null
let nextId = 0
let getGate: Promise<void> | null = null
let rejectWith: unknown = null

function usuario(overrides: Partial<StoredUsuario>): StoredUsuario {
  return {
    id: `usuario-${(nextId += 1)}`,
    email: 'ana@example.com',
    nombre: 'Ana',
    apellido: 'Pérez',
    rol: 'administrativo',
    perfilCompleto: true,
    fechaBaja: null,
    createdAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function installFetch() {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      const method = init?.method ?? 'GET'

      if (rejectWith) {
        throw rejectWith
      }

      if (failure) {
        return jsonResponse(failure.status, { code: failure.code ?? 'error', message: failure.message })
      }

      if (method === 'GET') {
        if (getGate) {
          await getGate
        }
        return jsonResponse(200, { usuarios: stored })
      }

      if (method === 'POST' && url.endsWith('/reactivar')) {
        const targetId = url.slice(0, -'/reactivar'.length).split('/').filter(Boolean).pop()
        stored = stored.map((candidate) =>
          candidate.id === targetId ? { ...candidate, fechaBaja: null } : candidate
        )
        return jsonResponse(200, stored.find((candidate) => candidate.id === targetId))
      }

      if (method === 'POST') {
        const values = JSON.parse(String(init?.body)) as Omit<
          StoredUsuario,
          'id' | 'perfilCompleto' | 'fechaBaja' | 'createdAt'
        >
        if (stored.some((candidate) => candidate.email === values.email)) {
          return jsonResponse(409, { code: 'email_in_use', message: 'El email ya está en uso' })
        }
        const created = usuario(values)
        stored = [...stored, created]
        return jsonResponse(201, { ...created, temporaryPassword: 'temp-pass-123' })
      }

      const id = url.slice(url.lastIndexOf('/') + 1).split('?')[0]

      if (method === 'PATCH') {
        const values = JSON.parse(String(init?.body)) as Partial<StoredUsuario>
        stored = stored.map((candidate) =>
          candidate.id === id ? { ...candidate, ...values } : candidate
        )
        return jsonResponse(200, stored.find((candidate) => candidate.id === id))
      }

      if (method === 'DELETE') {
        return new Response(null, { status: 204 })
      }

      return jsonResponse(405, { code: 'method_not_allowed', message: 'Método no permitido' })
    })
  )
}

function renderUsersPage() {
  return render(<UsersPage accessToken="token-123" />)
}

async function selectOption(user: ReturnType<typeof userEvent.setup>, triggerLabel: string, optionName: string) {
  await user.click(screen.getByRole('combobox', { name: triggerLabel }))
  await user.click(await screen.findByRole('option', { name: optionName }))
}

async function fillUserForm(
  user: ReturnType<typeof userEvent.setup>,
  values: { nombre: string; apellido: string; email?: string; rol?: string }
) {
  await user.type(screen.getByLabelText('Nombre'), values.nombre)
  await user.type(screen.getByLabelText('Apellido'), values.apellido)
  if (values.email) {
    await user.type(screen.getByLabelText('Correo electrónico'), values.email)
  }
  if (values.rol) {
    await selectOption(user, 'Rol', ROLE_LABELS[values.rol as GestionableRol])
  }
}

beforeEach(() => {
  stored = []
  failure = null
  nextId = 0
  getGate = null
  rejectWith = null
  installFetch()
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('UsersPage', () => {
  it('renders the section heading', () => {
    renderUsersPage()

    expect(screen.getByRole('heading', { name: 'Usuarios' })).toBeInTheDocument()
  })

  it('lists the users already stored in the backend', async () => {
    stored = [
      usuario({ nombre: 'Ana', apellido: 'Pérez', email: 'ana@example.com' }),
      usuario({ nombre: 'Luis', apellido: 'Gómez', email: 'luis@example.com', rol: 'escribano' }),
    ]
    renderUsersPage()

    expect(await screen.findByText('Ana Pérez')).toBeInTheDocument()
    expect(screen.getByText('Luis Gómez')).toBeInTheDocument()
  })

  it('shows an empty state when there are no users', async () => {
    renderUsersPage()

    expect(await screen.findByText('No hay usuarios cargados todavía.')).toBeInTheDocument()
  })

  it('shows the error returned by the backend when the list cannot be loaded', async () => {
    failure = { status: 403, message: 'Tu usuario no está habilitado para operar en el sistema' }
    renderUsersPage()

    expect(
      await screen.findByText('Tu usuario no está habilitado para operar en el sistema')
    ).toBeInTheDocument()
  })

  it('drops the pending load when the screen is unmounted', async () => {
    let releaseGet = () => {}
    getGate = new Promise<void>((resolve) => {
      releaseGet = resolve
    })
    stored = [usuario({ nombre: 'Ana', apellido: 'Pérez' })]

    const { unmount } = renderUsersPage()
    expect(screen.getByText('Cargando usuarios...')).toBeInTheDocument()

    unmount()
    releaseGet()
    await Promise.resolve()

    expect(screen.queryByText('Ana Pérez')).not.toBeInTheDocument()
  })

  it('shows a network error message when the list request fails to reach the server', async () => {
    rejectWith = { status: 500 }
    renderUsersPage()

    expect(
      await screen.findByText('No se pudo conectar con el servidor. Revisá tu conexión e intentá de nuevo.')
    ).toBeInTheDocument()
  })

  it('drops a pending load that fails after the screen is unmounted', async () => {
    let releaseGet = () => {}
    getGate = new Promise<void>((_resolve, reject) => {
      releaseGet = () => reject(new Error('boom'))
    })

    const { unmount } = renderUsersPage()
    expect(screen.getByText('Cargando usuarios...')).toBeInTheDocument()

    unmount()
    releaseGet()
    await Promise.resolve()

    expect(screen.queryByText('Ocurrió un error inesperado.')).not.toBeInTheDocument()
  })

  it('shows the backend error when deactivating a user fails', async () => {
    const user = userEvent.setup()
    stored = [usuario({ nombre: 'Ana', apellido: 'Pérez' })]
    renderUsersPage()
    await screen.findByText('Ana Pérez')

    await user.click(screen.getByRole('button', { name: 'Dar de baja' }))
    failure = { status: 409, message: 'El usuario ya está dado de baja' }
    await user.click(screen.getByRole('button', { name: 'Confirmar' }))

    expect(await screen.findByText('El usuario ya está dado de baja')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Confirmar' })).toBeInTheDocument()
  })

  it('shows the backend error when editing a user fails', async () => {
    const user = userEvent.setup()
    stored = [usuario({ nombre: 'Ana', apellido: 'Pérez' })]
    renderUsersPage()
    await screen.findByText('Ana Pérez')

    await user.click(screen.getByRole('button', { name: 'Editar' }))
    failure = { status: 409, message: 'No se enviaron campos para modificar' }
    await user.click(screen.getByRole('button', { name: 'Guardar cambios' }))

    expect(
      await screen.findByText('No se enviaron campos para modificar')
    ).toBeInTheDocument()
  })

  it('creates a new user and shows its temporary password', async () => {
    const user = userEvent.setup()
    renderUsersPage()
    await screen.findByText('No hay usuarios cargados todavía.')

    await user.click(screen.getByRole('button', { name: 'Nuevo usuario' }))
    await fillUserForm(user, {
      nombre: 'Ana',
      apellido: 'Pérez',
      email: 'ana@example.com',
      rol: 'escribano',
    })
    await user.click(screen.getByRole('button', { name: 'Crear usuario' }))

    const card = (await screen.findByText('Ana Pérez')).closest('li') as HTMLElement
    expect(within(card).getByText('Escribano')).toBeInTheDocument()
    expect(screen.getByText('temp-pass-123')).toBeInTheDocument()
  })

  it('requires nombre, apellido and email', async () => {
    const user = userEvent.setup()
    renderUsersPage()
    await screen.findByText('No hay usuarios cargados todavía.')

    await user.click(screen.getByRole('button', { name: 'Nuevo usuario' }))
    await user.click(screen.getByRole('button', { name: 'Crear usuario' }))

    expect(
      await screen.findByText('Completá nombre, apellido y correo electrónico.')
    ).toBeInTheDocument()
  })

  it('rejects a duplicate email before reaching the backend', async () => {
    const user = userEvent.setup()
    stored = [usuario({ email: 'ana@example.com' })]
    renderUsersPage()
    await screen.findByText('Ana Pérez')

    await user.click(screen.getByRole('button', { name: 'Nuevo usuario' }))
    await fillUserForm(user, { nombre: 'Luis', apellido: 'Gómez', email: 'ana@example.com' })
    await user.click(screen.getByRole('button', { name: 'Crear usuario' }))

    expect(
      await screen.findByText('Ya existe un usuario con ese correo electrónico.')
    ).toBeInTheDocument()
    expect(screen.queryByText('Luis Gómez')).not.toBeInTheDocument()
  })

  it('shows the backend error when the alta is rejected', async () => {
    const user = userEvent.setup()
    renderUsersPage()
    await screen.findByText('No hay usuarios cargados todavía.')

    await user.click(screen.getByRole('button', { name: 'Nuevo usuario' }))
    await fillUserForm(user, { nombre: 'Ana', apellido: 'Pérez', email: 'ana@example.com' })
    failure = { status: 409, message: 'El email ya está en uso' }
    await user.click(screen.getByRole('button', { name: 'Crear usuario' }))

    expect(await screen.findByText('El email ya está en uso')).toBeInTheDocument()
  })

  it('edits an existing user, keeping its email and rol read-only', async () => {
    const user = userEvent.setup()
    stored = [
      usuario({ nombre: 'Ana', apellido: 'Pérez', email: 'ana@example.com' }),
      usuario({ nombre: 'Luis', apellido: 'Gómez', email: 'luis@example.com' }),
    ]
    renderUsersPage()
    await screen.findByText('Ana Pérez')

    await user.click(screen.getAllByRole('button', { name: 'Editar' })[0])
    expect(screen.getByText('ana@example.com · Administrativo')).toBeInTheDocument()
    expect(screen.queryByLabelText('Correo electrónico')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Rol')).not.toBeInTheDocument()

    const nombreInput = screen.getByLabelText('Nombre')
    await user.clear(nombreInput)
    await user.type(nombreInput, 'Ana María')
    await user.click(screen.getByRole('button', { name: 'Guardar cambios' }))

    expect(await screen.findByText('Ana María Pérez')).toBeInTheDocument()
    expect(screen.getByText('Luis Gómez')).toBeInTheDocument()
  })

  it('deactivates a user after confirming the baja inline', async () => {
    const user = userEvent.setup()
    stored = [
      usuario({ nombre: 'Ana', apellido: 'Pérez' }),
      usuario({ nombre: 'Luis', apellido: 'Gómez' }),
    ]
    renderUsersPage()
    await screen.findByText('Ana Pérez')

    await user.click(screen.getAllByRole('button', { name: 'Dar de baja' })[0])
    expect(screen.getByText('¿Confirmar baja?')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Confirmar' }))

    expect(await screen.findByText('Dado de baja')).toBeInTheDocument()
    expect(screen.getAllByText('Activo')).toHaveLength(1)
    expect(screen.getAllByRole('button', { name: 'Dar de baja' })).toHaveLength(1)
    expect(screen.getAllByRole('button', { name: 'Editar' })).toHaveLength(1)
  })

  it('reconciles the baja locally when a retry lands on an already-inactive conflict', async () => {
    const user = userEvent.setup()
    stored = [usuario({ nombre: 'Ana', apellido: 'Pérez' })]
    renderUsersPage()
    await screen.findByText('Ana Pérez')

    await user.click(screen.getByRole('button', { name: 'Dar de baja' }))
    failure = { status: 409, code: 'user_already_inactive', message: 'El usuario ya está dado de baja' }
    await user.click(screen.getByRole('button', { name: 'Confirmar' }))

    expect(await screen.findByText('Dado de baja')).toBeInTheDocument()
    expect(screen.queryByText('El usuario ya está dado de baja')).not.toBeInTheDocument()
  })

  it('reconciles a reactivación locally when a retry lands on an already-active conflict', async () => {
    const user = userEvent.setup()
    stored = [{ ...usuario({ nombre: 'Ana', apellido: 'Pérez' }), fechaBaja: '2026-01-01T00:00:00Z' }]
    renderUsersPage()
    await screen.findByText('Ana Pérez')

    failure = { status: 409, code: 'user_already_active', message: 'El usuario ya está activo' }
    await user.click(screen.getByRole('button', { name: 'Reactivar' }))

    expect(await screen.findByText('Activo')).toBeInTheDocument()
    expect(screen.queryByText('El usuario ya está activo')).not.toBeInTheDocument()
  })

  it('keeps the user when the inline baja confirmation is cancelled', async () => {
    const user = userEvent.setup()
    stored = [usuario({ nombre: 'Ana', apellido: 'Pérez' })]
    renderUsersPage()
    await screen.findByText('Ana Pérez')

    await user.click(screen.getByRole('button', { name: 'Dar de baja' }))
    await user.click(screen.getByRole('button', { name: 'Cancelar' }))

    expect(screen.getByRole('button', { name: 'Dar de baja' })).toBeInTheDocument()
    expect(screen.queryByText('¿Confirmar baja?')).not.toBeInTheDocument()
  })

  it('reactivates a user given de baja', async () => {
    const user = userEvent.setup()
    stored = [
      usuario({ nombre: 'Ana', apellido: 'Pérez', fechaBaja: '2026-01-15T00:00:00Z' }),
      usuario({ nombre: 'Luis', apellido: 'Gómez' }),
    ]
    renderUsersPage()
    await screen.findByText('Ana Pérez')

    expect(screen.getByText('Dado de baja')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Reactivar' }))

    expect(await screen.findAllByText('Activo')).toHaveLength(2)
    expect(screen.queryByRole('button', { name: 'Reactivar' })).not.toBeInTheDocument()
    expect(screen.getAllByRole('button', { name: 'Editar' })).toHaveLength(2)
    expect(screen.getAllByRole('button', { name: 'Dar de baja' })).toHaveLength(2)
  })

  it('shows the backend error when reactivating a user fails', async () => {
    const user = userEvent.setup()
    stored = [usuario({ nombre: 'Ana', apellido: 'Pérez', fechaBaja: '2026-01-15T00:00:00Z' })]
    renderUsersPage()
    await screen.findByText('Ana Pérez')

    failure = { status: 409, message: 'El usuario ya está activo' }
    await user.click(screen.getByRole('button', { name: 'Reactivar' }))

    expect(await screen.findByText('El usuario ya está activo')).toBeInTheDocument()
    expect(screen.getByText('Dado de baja')).toBeInTheDocument()
  })

  it('filters the list by rol', async () => {
    const user = userEvent.setup()
    stored = [
      usuario({ nombre: 'Ana', apellido: 'Pérez', rol: 'administrativo' }),
      usuario({ nombre: 'Luis', apellido: 'Gómez', rol: 'escribano' }),
    ]
    renderUsersPage()
    await screen.findByText('Ana Pérez')

    await selectOption(user, 'Rol', 'Escribano')

    expect(screen.queryByText('Ana Pérez')).not.toBeInTheDocument()
    expect(screen.getByText('Luis Gómez')).toBeInTheDocument()
  })

  it('creates and filters by the agrimensor role', async () => {
    const user = userEvent.setup()
    renderUsersPage()
    await screen.findByText('No hay usuarios cargados todavía.')

    await user.click(screen.getByRole('button', { name: 'Nuevo usuario' }))
    await fillUserForm(user, {
      nombre: 'Mara',
      apellido: 'Cruz',
      email: 'mara@example.com',
      rol: 'agrimensor',
    })
    await user.click(screen.getByRole('button', { name: 'Crear usuario' }))

    const card = (await screen.findByText('Mara Cruz')).closest('li') as HTMLElement
    expect(within(card).getByText('Agrimensor')).toBeInTheDocument()

    await selectOption(user, 'Rol', 'Agrimensor')

    expect(screen.getByText('Mara Cruz')).toBeInTheDocument()
  })

  it('filters the list by estado', async () => {
    const user = userEvent.setup()
    stored = [
      usuario({ nombre: 'Ana', apellido: 'Pérez', fechaBaja: null }),
      usuario({ nombre: 'Luis', apellido: 'Gómez', fechaBaja: '2026-01-15T00:00:00Z' }),
    ]
    renderUsersPage()
    await screen.findByText('Ana Pérez')
    expect(screen.getByText('Luis Gómez')).toBeInTheDocument()

    await selectOption(user, 'Estado', 'Activos')

    expect(screen.getByText('Ana Pérez')).toBeInTheDocument()
    expect(screen.queryByText('Luis Gómez')).not.toBeInTheDocument()

    await selectOption(user, 'Estado', 'Dados de baja')

    expect(screen.queryByText('Ana Pérez')).not.toBeInTheDocument()
    expect(screen.getByText('Luis Gómez')).toBeInTheDocument()
  })

  it('filters the list by search text, matching nombre, apellido or email', async () => {
    const user = userEvent.setup()
    stored = [
      usuario({ nombre: 'Ana', apellido: 'Pérez', email: 'ana@example.com' }),
      usuario({ nombre: 'Luis', apellido: 'Gómez', email: 'luis@example.com' }),
    ]
    renderUsersPage()
    await screen.findByText('Ana Pérez')

    await user.type(screen.getByLabelText('Buscar'), 'gomez')

    expect(screen.queryByText('Ana Pérez')).not.toBeInTheDocument()
    expect(screen.getByText('Luis Gómez')).toBeInTheDocument()

    await user.clear(screen.getByLabelText('Buscar'))
    await user.type(screen.getByLabelText('Buscar'), 'ana@example.com')

    expect(screen.getByText('Ana Pérez')).toBeInTheDocument()
    expect(screen.queryByText('Luis Gómez')).not.toBeInTheDocument()
  })

  it('ignores accents and casing when filtering by search text', async () => {
    const user = userEvent.setup()
    stored = [usuario({ nombre: 'Ana', apellido: 'Pérez', email: 'ana@example.com' })]
    renderUsersPage()
    await screen.findByText('Ana Pérez')

    await user.type(screen.getByLabelText('Buscar'), 'PEREZ')

    expect(screen.getByText('Ana Pérez')).toBeInTheDocument()
  })

  it('combines the search text with the rol and estado filters', async () => {
    const user = userEvent.setup()
    stored = [
      usuario({ nombre: 'Ana', apellido: 'Pérez', rol: 'administrativo' }),
      usuario({ nombre: 'Ana', apellido: 'Gómez', rol: 'escribano' }),
    ]
    renderUsersPage()
    await screen.findAllByText(/^Ana /)

    await user.type(screen.getByLabelText('Buscar'), 'ana')
    await selectOption(user, 'Rol', 'Escribano')

    expect(screen.queryByText('Ana Pérez')).not.toBeInTheDocument()
    expect(screen.getByText('Ana Gómez')).toBeInTheDocument()
  })

  it('shows a message when the filters have no matches', async () => {
    const user = userEvent.setup()
    stored = [usuario({ nombre: 'Ana', apellido: 'Pérez', rol: 'administrativo' })]
    renderUsersPage()
    await screen.findByText('Ana Pérez')

    await selectOption(user, 'Rol', 'Escribano')

    expect(screen.getByText('No se encontraron usuarios con esos filtros.')).toBeInTheDocument()
  })

  it('closes the form without saving when cancel is clicked', async () => {
    const user = userEvent.setup()
    renderUsersPage()
    await screen.findByText('No hay usuarios cargados todavía.')

    await user.click(screen.getByRole('button', { name: 'Nuevo usuario' }))
    await fillUserForm(user, { nombre: 'Ana', apellido: 'Pérez', email: 'ana@example.com' })
    await user.click(screen.getByRole('button', { name: 'Cancelar' }))

    expect(screen.queryByText('Ana Pérez')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Nuevo usuario' })).toBeInTheDocument()
  })

  it('does not show filters when there are no users', async () => {
    renderUsersPage()
    await screen.findByText('No hay usuarios cargados todavía.')

    expect(screen.queryByLabelText('Buscar')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Rol')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Estado')).not.toBeInTheDocument()
  })

  it('hides the list while a form is open', async () => {
    const user = userEvent.setup()
    stored = [usuario({ nombre: 'Ana', apellido: 'Pérez' })]
    renderUsersPage()
    await screen.findByText('Ana Pérez')

    await user.click(screen.getByRole('button', { name: 'Nuevo usuario' }))

    expect(screen.queryByText('Ana Pérez')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Buscar')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Estado')).not.toBeInTheDocument()
  })
})
