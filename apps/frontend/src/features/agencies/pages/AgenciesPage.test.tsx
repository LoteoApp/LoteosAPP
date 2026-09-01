import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Session, User } from '@supabase/supabase-js'
import AgenciesPage from './AgenciesPage'
import { AuthContext, type AuthContextValue } from '../../auth/hooks/use-auth'

type StoredInmobiliaria = {
  id: string
  razonSocial: string
  cuit?: string
  telefono?: string
  email?: string
}

let stored: StoredInmobiliaria[] = []
let failure: { status: number; message: string } | null = null
let nextId = 0
let postCalls = 0
let getGate: Promise<void> | null = null
let postGate: Promise<void> | null = null
let rejectWith: unknown = null

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
        return jsonResponse(failure.status, { code: 'error', message: failure.message })
      }

      if (method === 'GET') {
        if (getGate) {
          await getGate
        }
        return jsonResponse(200, { inmobiliarias: stored })
      }

      if (method === 'POST') {
        postCalls += 1
        if (postGate) {
          await postGate
        }
        const values = JSON.parse(String(init?.body)) as Omit<StoredInmobiliaria, 'id'>
        nextId += 1
        const created = { id: `inmobiliaria-${nextId}`, ...values }
        stored = [...stored, created]
        return jsonResponse(201, created)
      }

      const id = url.slice(url.lastIndexOf('/') + 1)

      if (method === 'PATCH') {
        const values = JSON.parse(String(init?.body)) as Partial<StoredInmobiliaria>
        stored = stored.map((inmobiliaria) =>
          inmobiliaria.id === id ? { ...inmobiliaria, ...values } : inmobiliaria,
        )
        return jsonResponse(200, stored.find((inmobiliaria) => inmobiliaria.id === id))
      }

      if (method === 'DELETE') {
        stored = stored.filter((inmobiliaria) => inmobiliaria.id !== id)
        return new Response(null, { status: 204 })
      }

      return jsonResponse(405, { code: 'method_not_allowed', message: 'Método no permitido' })
    }),
  )
}

function renderAgenciesPage(role: string | null = 'administrador') {
  const value: AuthContextValue = {
    isLoading: false,
    session: { access_token: 'token-123' } as unknown as Session,
    user: role ? ({ app_metadata: { role } } as unknown as User) : null,
    error: null,
    login: vi.fn(),
    logout: vi.fn(),
  }

  return render(
    <AuthContext.Provider value={value}>
      <AgenciesPage />
    </AuthContext.Provider>,
  )
}

async function fillAgencyForm(
  user: ReturnType<typeof userEvent.setup>,
  values: { razonSocial: string; cuit?: string; telefono?: string; email?: string },
) {
  await user.type(screen.getByLabelText('Razón social'), values.razonSocial)
  if (values.cuit) {
    await user.type(screen.getByLabelText('CUIT'), values.cuit)
  }
  if (values.telefono) {
    await user.type(screen.getByLabelText('Teléfono'), values.telefono)
  }
  if (values.email) {
    await user.type(screen.getByLabelText('Correo electrónico'), values.email)
  }
}

beforeEach(() => {
  stored = []
  failure = null
  nextId = 0
  postCalls = 0
  getGate = null
  postGate = null
  rejectWith = null
  installFetch()
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('AgenciesPage', () => {
  it('renders the section heading', () => {
    renderAgenciesPage()

    expect(screen.getByRole('heading', { name: 'Inmobiliarias' })).toBeInTheDocument()
  })

  it('lists the agencies already stored in the backend', async () => {
    stored = [
      { id: 'inmobiliaria-1', razonSocial: 'Lotes del Sur', cuit: '30712345678' },
      { id: 'inmobiliaria-2', razonSocial: 'Altamira Propiedades' },
    ]
    renderAgenciesPage()

    expect(await screen.findByText('Lotes del Sur')).toBeInTheDocument()
    expect(screen.getByText('Altamira Propiedades')).toBeInTheDocument()
    expect(screen.getByText('CUIT 30712345678')).toBeInTheDocument()
  })

  it('shows the contact data of an agency that has it', async () => {
    stored = [
      {
        id: 'inmobiliaria-1',
        razonSocial: 'Lotes del Sur',
        telefono: '3415551234',
        email: 'contacto@lotesdelsur.com',
      },
    ]
    renderAgenciesPage()

    expect(
      await screen.findByText('3415551234 · contacto@lotesdelsur.com'),
    ).toBeInTheDocument()
  })

  it('shows a loading message while the list is in flight', async () => {
    let release = () => {}
    getGate = new Promise<void>((resolve) => {
      release = resolve
    })
    renderAgenciesPage()

    expect(screen.getByText('Cargando inmobiliarias...')).toBeInTheDocument()

    release()
    expect(await screen.findByText('No hay inmobiliarias cargadas todavía.')).toBeInTheDocument()
  })

  it('shows an empty state when there are no agencies', async () => {
    renderAgenciesPage()

    expect(await screen.findByText('No hay inmobiliarias cargadas todavía.')).toBeInTheDocument()
  })

  it('surfaces the error of a failed load', async () => {
    failure = { status: 503, message: 'La base de datos no está disponible' }
    renderAgenciesPage()

    expect(await screen.findByText('La base de datos no está disponible')).toBeInTheDocument()
  })

  it('does not claim the list is empty when it failed to load', async () => {
    failure = { status: 403, message: 'No tenés permisos para esta acción' }
    renderAgenciesPage()

    await screen.findByText('No tenés permisos para esta acción')

    expect(
      screen.queryByText('No hay inmobiliarias cargadas todavía.'),
    ).not.toBeInTheDocument()
  })

  it('shows a generic message when the failure carries no message', async () => {
    rejectWith = { status: 500 }
    renderAgenciesPage()

    expect(await screen.findByText('Ocurrió un error inesperado.')).toBeInTheDocument()
  })

  it('drops the pending load when the screen is unmounted', async () => {
    let releaseGet = () => {}
    getGate = new Promise<void>((resolve) => {
      releaseGet = resolve
    })
    stored = [{ id: 'inmobiliaria-1', razonSocial: 'Lotes del Sur' }]

    const { unmount } = renderAgenciesPage()
    expect(screen.getByText('Cargando inmobiliarias...')).toBeInTheDocument()

    unmount()
    releaseGet()
    await Promise.resolve()

    expect(screen.queryByText('Lotes del Sur')).not.toBeInTheDocument()
  })

  it('sends a single request when the alta is submitted twice in a row', async () => {
    const user = userEvent.setup()
    let releasePost = () => {}
    postGate = new Promise<void>((resolve) => {
      releasePost = resolve
    })
    renderAgenciesPage()
    await screen.findByText('No hay inmobiliarias cargadas todavía.')

    await user.click(screen.getByRole('button', { name: 'Nueva inmobiliaria' }))
    await fillAgencyForm(user, { razonSocial: 'Lotes del Sur' })

    const submit = screen.getByRole('button', { name: 'Crear inmobiliaria' })
    await user.click(submit)

    expect(await screen.findByRole('button', { name: 'Guardando...' })).toBeDisabled()

    await user.click(submit)
    releasePost()

    expect(await screen.findByText('Lotes del Sur')).toBeInTheDocument()
    expect(postCalls).toBe(1)
  })

  it('creates an agency and shows it in the list', async () => {
    const user = userEvent.setup()
    renderAgenciesPage()
    await screen.findByText('No hay inmobiliarias cargadas todavía.')

    await user.click(screen.getByRole('button', { name: 'Nueva inmobiliaria' }))
    await fillAgencyForm(user, {
      razonSocial: 'Lotes del Sur',
      cuit: '30-71234567-8',
      telefono: '3415551234',
      email: 'contacto@lotesdelsur.com',
    })
    await user.click(screen.getByRole('button', { name: 'Crear inmobiliaria' }))

    expect(await screen.findByText('Lotes del Sur')).toBeInTheDocument()
  })

  it('rejects an agency without razón social', async () => {
    const user = userEvent.setup()
    renderAgenciesPage()
    await screen.findByText('No hay inmobiliarias cargadas todavía.')

    await user.click(screen.getByRole('button', { name: 'Nueva inmobiliaria' }))
    await user.click(screen.getByRole('button', { name: 'Crear inmobiliaria' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Completá la razón social.')
  })

  it('rejects a CUIT that is not eleven digits', async () => {
    const user = userEvent.setup()
    renderAgenciesPage()
    await screen.findByText('No hay inmobiliarias cargadas todavía.')

    await user.click(screen.getByRole('button', { name: 'Nueva inmobiliaria' }))
    await fillAgencyForm(user, { razonSocial: 'Lotes del Sur', cuit: '3071234' })
    await user.click(screen.getByRole('button', { name: 'Crear inmobiliaria' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'El CUIT debe tener 11 dígitos.',
    )
  })

  it('rejects a CUIT another agency already uses, however it is typed', async () => {
    stored = [{ id: 'inmobiliaria-1', razonSocial: 'Altamira', cuit: '30712345678' }]
    const user = userEvent.setup()
    renderAgenciesPage()
    await screen.findByText('Altamira')

    await user.click(screen.getByRole('button', { name: 'Nueva inmobiliaria' }))
    await fillAgencyForm(user, { razonSocial: 'Lotes del Sur', cuit: '30-71234567-8' })
    await user.click(screen.getByRole('button', { name: 'Crear inmobiliaria' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Ya existe una inmobiliaria con ese CUIT.',
    )
  })

  it('keeps the form open and shows the error when the backend rejects the alta', async () => {
    const user = userEvent.setup()
    renderAgenciesPage()
    await screen.findByText('No hay inmobiliarias cargadas todavía.')

    await user.click(screen.getByRole('button', { name: 'Nueva inmobiliaria' }))
    await fillAgencyForm(user, { razonSocial: 'Lotes del Sur' })
    failure = { status: 409, message: 'El CUIT ya está en uso' }
    await user.click(screen.getByRole('button', { name: 'Crear inmobiliaria' }))

    expect(await screen.findByText('El CUIT ya está en uso')).toBeInTheDocument()
    expect(screen.getByRole('form', { name: 'Datos de la inmobiliaria' })).toBeInTheDocument()
  })

  it('closes the form when the alta is cancelled', async () => {
    const user = userEvent.setup()
    renderAgenciesPage()
    await screen.findByText('No hay inmobiliarias cargadas todavía.')

    await user.click(screen.getByRole('button', { name: 'Nueva inmobiliaria' }))
    await user.click(screen.getByRole('button', { name: 'Cancelar' }))

    expect(screen.getByRole('button', { name: 'Nueva inmobiliaria' })).toBeInTheDocument()
  })

  it('edits an existing agency', async () => {
    stored = [{ id: 'inmobiliaria-1', razonSocial: 'Lotes del Sur' }]
    const user = userEvent.setup()
    renderAgenciesPage()
    await screen.findByText('Lotes del Sur')

    await user.click(screen.getByRole('button', { name: 'Editar' }))
    const razonSocial = screen.getByLabelText('Razón social')
    await user.clear(razonSocial)
    await user.type(razonSocial, 'Lotes del Sur SRL')
    await user.click(screen.getByRole('button', { name: 'Guardar cambios' }))

    expect(await screen.findByText('Lotes del Sur SRL')).toBeInTheDocument()
  })

  it('gives an agency de baja after confirming', async () => {
    stored = [{ id: 'inmobiliaria-1', razonSocial: 'Lotes del Sur' }]
    const user = userEvent.setup()
    renderAgenciesPage()
    await screen.findByText('Lotes del Sur')

    await user.click(screen.getByRole('button', { name: 'Dar de baja' }))
    await user.click(screen.getByRole('button', { name: 'Confirmar' }))

    expect(
      await screen.findByText('No hay inmobiliarias cargadas todavía.'),
    ).toBeInTheDocument()
  })

  it('keeps the agency when the baja is cancelled', async () => {
    stored = [{ id: 'inmobiliaria-1', razonSocial: 'Lotes del Sur' }]
    const user = userEvent.setup()
    renderAgenciesPage()
    await screen.findByText('Lotes del Sur')

    await user.click(screen.getByRole('button', { name: 'Dar de baja' }))
    await user.click(screen.getByRole('button', { name: 'Cancelar' }))

    expect(screen.getByText('Lotes del Sur')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Dar de baja' })).toBeInTheDocument()
  })

  it('filters the list by razón social, ignoring accents', async () => {
    stored = [
      { id: 'inmobiliaria-1', razonSocial: 'Río Paraná Inmuebles' },
      { id: 'inmobiliaria-2', razonSocial: 'Altamira Propiedades' },
    ]
    const user = userEvent.setup()
    renderAgenciesPage()
    await screen.findByText('Río Paraná Inmuebles')

    await user.type(screen.getByLabelText('Buscar'), 'rio')

    expect(screen.getByText('Río Paraná Inmuebles')).toBeInTheDocument()
    expect(screen.queryByText('Altamira Propiedades')).not.toBeInTheDocument()
  })

  it('filters the list by CUIT however it is typed', async () => {
    stored = [
      { id: 'inmobiliaria-1', razonSocial: 'Lotes del Sur', cuit: '30712345678' },
      { id: 'inmobiliaria-2', razonSocial: 'Altamira Propiedades', cuit: '30999999999' },
    ]
    const user = userEvent.setup()
    renderAgenciesPage()
    await screen.findByText('Lotes del Sur')

    await user.type(screen.getByLabelText('Buscar'), '30-71234567-8')

    expect(screen.getByText('Lotes del Sur')).toBeInTheDocument()
    expect(screen.queryByText('Altamira Propiedades')).not.toBeInTheDocument()
  })

  it('reports a search with no matches', async () => {
    stored = [{ id: 'inmobiliaria-1', razonSocial: 'Lotes del Sur' }]
    const user = userEvent.setup()
    renderAgenciesPage()
    await screen.findByText('Lotes del Sur')

    await user.type(screen.getByLabelText('Buscar'), 'altamira')

    expect(
      screen.getByText('No se encontraron inmobiliarias con esa búsqueda.'),
    ).toBeInTheDocument()
  })

  it('hides every write action from a non administrador', async () => {
    stored = [{ id: 'inmobiliaria-1', razonSocial: 'Lotes del Sur' }]
    renderAgenciesPage('administrativo')

    const item = await screen.findByRole('listitem')
    expect(within(item).queryByRole('button')).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Nueva inmobiliaria' }),
    ).not.toBeInTheDocument()
  })
})
