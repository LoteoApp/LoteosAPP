import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Session, User } from '@supabase/supabase-js'
import SurveyorsPage from './SurveyorsPage'
import { AuthContext, type AuthContextValue } from '../../auth/hooks/use-auth'
import {
  createSurveyor,
  deactivateSurveyor,
  listSurveyors,
  updateSurveyor,
} from '../api/surveyors'
import type { Surveyor } from '../types'

vi.mock('../api/surveyors', () => ({
  listSurveyors: vi.fn(),
  createSurveyor: vi.fn(),
  updateSurveyor: vi.fn(),
  deactivateSurveyor: vi.fn(),
}))

const listSurveyorsMock = vi.mocked(listSurveyors)
const createSurveyorMock = vi.mocked(createSurveyor)
const updateSurveyorMock = vi.mocked(updateSurveyor)
const deactivateSurveyorMock = vi.mocked(deactivateSurveyor)

const ana: Surveyor = {
  id: 'agri-1',
  nombre: 'Ana',
  apellido: 'Gómez',
  email: 'ana@example.com',
  fechaBaja: null,
}

const luis: Surveyor = {
  id: 'agri-2',
  nombre: 'Luis',
  apellido: 'Paz',
  email: 'luis@example.com',
  fechaBaja: '2026-03-01T12:00:00Z',
}

function renderSurveyorsPage(role: string | null = 'administrador') {
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
      <SurveyorsPage />
    </AuthContext.Provider>,
  )
}

async function fillForm(
  user: ReturnType<typeof userEvent.setup>,
  values: { nombre: string; apellido: string; email?: string },
) {
  await user.type(screen.getByLabelText('Nombre'), values.nombre)
  await user.type(screen.getByLabelText('Apellido'), values.apellido)
  if (values.email !== undefined) {
    await user.type(screen.getByLabelText('Correo electrónico'), values.email)
  }
}

beforeEach(() => {
  listSurveyorsMock.mockReset().mockResolvedValue([])
  createSurveyorMock.mockReset()
  updateSurveyorMock.mockReset()
  deactivateSurveyorMock.mockReset()
})

describe('SurveyorsPage', () => {
  it('shows an empty state when there are no surveyors', async () => {
    renderSurveyorsPage()

    expect(
      await screen.findByText('No hay agrimensores cargados todavía.'),
    ).toBeInTheDocument()
  })

  it('lists the surveyors with their status', async () => {
    listSurveyorsMock.mockResolvedValue([ana, luis])
    renderSurveyorsPage()

    expect(await screen.findByText('Ana Gómez')).toBeInTheDocument()
    expect(screen.getByText('ana@example.com')).toBeInTheDocument()
    expect(screen.getByText('Activo')).toBeInTheDocument()
    expect(screen.getByText('Luis Paz')).toBeInTheDocument()
    expect(screen.getByText('Dado de baja')).toBeInTheDocument()
  })

  it('hides the edit and baja actions for a surveyor already given de baja', async () => {
    listSurveyorsMock.mockResolvedValue([luis])
    renderSurveyorsPage()

    expect(await screen.findByText('Luis Paz')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Editar' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Dar de baja' })).not.toBeInTheDocument()
  })

  it('asks for the surveyors given de baja when the filter is checked', async () => {
    const user = userEvent.setup()
    renderSurveyorsPage()

    await screen.findByText('No hay agrimensores cargados todavía.')
    listSurveyorsMock.mockResolvedValue([luis])

    await user.click(screen.getByLabelText('Mostrar dados de baja'))

    expect(await screen.findByText('Luis Paz')).toBeInTheDocument()
    expect(listSurveyorsMock).toHaveBeenLastCalledWith('token-123', true)
  })

  it('creates a surveyor and shows its temporary password', async () => {
    const user = userEvent.setup()
    createSurveyorMock.mockResolvedValue({ surveyor: ana, temporaryPassword: 'temp-pass-123' })
    renderSurveyorsPage()

    await screen.findByText('No hay agrimensores cargados todavía.')
    await user.click(screen.getByRole('button', { name: 'Nuevo agrimensor' }))
    await fillForm(user, { nombre: 'Ana', apellido: 'Gómez', email: 'ana@example.com' })

    listSurveyorsMock.mockResolvedValue([ana])
    await user.click(screen.getByRole('button', { name: 'Crear agrimensor' }))

    expect(await screen.findByText('Contraseña temporal: temp-pass-123')).toBeInTheDocument()
    expect(screen.getByText('Ana Gómez')).toBeInTheDocument()
    expect(createSurveyorMock).toHaveBeenCalledWith('token-123', {
      nombre: 'Ana',
      apellido: 'Gómez',
      email: 'ana@example.com',
    })
  })

  it('dismisses the temporary password notice', async () => {
    const user = userEvent.setup()
    createSurveyorMock.mockResolvedValue({ surveyor: ana, temporaryPassword: 'temp-pass-123' })
    renderSurveyorsPage()

    await screen.findByText('No hay agrimensores cargados todavía.')
    await user.click(screen.getByRole('button', { name: 'Nuevo agrimensor' }))
    await fillForm(user, { nombre: 'Ana', apellido: 'Gómez', email: 'ana@example.com' })
    await user.click(screen.getByRole('button', { name: 'Crear agrimensor' }))

    await screen.findByText('Contraseña temporal: temp-pass-123')
    await user.click(screen.getByRole('button', { name: 'Entendido' }))

    expect(screen.queryByText('Contraseña temporal: temp-pass-123')).not.toBeInTheDocument()
  })

  it('requires nombre and apellido', async () => {
    const user = userEvent.setup()
    renderSurveyorsPage()

    await screen.findByText('No hay agrimensores cargados todavía.')
    await user.click(screen.getByRole('button', { name: 'Nuevo agrimensor' }))
    await user.click(screen.getByRole('button', { name: 'Crear agrimensor' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Completá nombre y apellido.')
    expect(createSurveyorMock).not.toHaveBeenCalled()
  })

  it('requires a valid email on the alta', async () => {
    const user = userEvent.setup()
    renderSurveyorsPage()

    await screen.findByText('No hay agrimensores cargados todavía.')
    await user.click(screen.getByRole('button', { name: 'Nuevo agrimensor' }))
    await fillForm(user, { nombre: 'Ana', apellido: 'Gómez', email: 'sin-arroba' })
    await user.click(screen.getByRole('button', { name: 'Crear agrimensor' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Ingresá un correo electrónico válido.',
    )
    expect(createSurveyorMock).not.toHaveBeenCalled()
  })

  it('shows the API error when the alta fails', async () => {
    const user = userEvent.setup()
    createSurveyorMock.mockRejectedValue(new Error('El email ya está en uso'))
    renderSurveyorsPage()

    await screen.findByText('No hay agrimensores cargados todavía.')
    await user.click(screen.getByRole('button', { name: 'Nuevo agrimensor' }))
    await fillForm(user, { nombre: 'Ana', apellido: 'Gómez', email: 'ana@example.com' })
    await user.click(screen.getByRole('button', { name: 'Crear agrimensor' }))

    expect(await screen.findByText('El email ya está en uso')).toBeInTheDocument()
  })

  it('edits a surveyor without letting the email change', async () => {
    const user = userEvent.setup()
    listSurveyorsMock.mockResolvedValue([ana])
    updateSurveyorMock.mockResolvedValue({ ...ana, nombre: 'Ana María' })
    renderSurveyorsPage()

    await screen.findByText('Ana Gómez')
    await user.click(screen.getByRole('button', { name: 'Editar' }))

    expect(screen.getByLabelText('Correo electrónico')).toBeDisabled()

    const nombre = screen.getByLabelText('Nombre')
    await user.clear(nombre)
    await user.type(nombre, 'Ana María')

    listSurveyorsMock.mockResolvedValue([{ ...ana, nombre: 'Ana María' }])
    await user.click(screen.getByRole('button', { name: 'Guardar cambios' }))

    expect(await screen.findByText('Ana María Gómez')).toBeInTheDocument()
    expect(updateSurveyorMock).toHaveBeenCalledWith('token-123', 'agri-1', {
      nombre: 'Ana María',
      apellido: 'Gómez',
    })
  })

  it('closes the form without saving when cancel is clicked', async () => {
    const user = userEvent.setup()
    listSurveyorsMock.mockResolvedValue([ana])
    renderSurveyorsPage()

    await screen.findByText('Ana Gómez')
    await user.click(screen.getByRole('button', { name: 'Editar' }))
    await user.click(screen.getByRole('button', { name: 'Cancelar' }))

    expect(updateSurveyorMock).not.toHaveBeenCalled()
    expect(screen.getByRole('button', { name: 'Nuevo agrimensor' })).toBeInTheDocument()
  })

  it('gives a surveyor de baja after confirming inline', async () => {
    const user = userEvent.setup()
    listSurveyorsMock.mockResolvedValue([ana])
    deactivateSurveyorMock.mockResolvedValue(undefined)
    renderSurveyorsPage()

    await screen.findByText('Ana Gómez')
    await user.click(screen.getByRole('button', { name: 'Dar de baja' }))
    expect(screen.getByText('¿Confirmar baja?')).toBeInTheDocument()

    listSurveyorsMock.mockResolvedValue([])
    await user.click(screen.getByRole('button', { name: 'Confirmar' }))

    await waitFor(() => {
      expect(screen.queryByText('Ana Gómez')).not.toBeInTheDocument()
    })
    expect(deactivateSurveyorMock).toHaveBeenCalledWith('token-123', 'agri-1')
  })

  it('keeps the surveyor when the baja confirmation is cancelled', async () => {
    const user = userEvent.setup()
    listSurveyorsMock.mockResolvedValue([ana])
    renderSurveyorsPage()

    await screen.findByText('Ana Gómez')
    await user.click(screen.getByRole('button', { name: 'Dar de baja' }))
    await user.click(screen.getByRole('button', { name: 'Cancelar' }))

    expect(deactivateSurveyorMock).not.toHaveBeenCalled()
    expect(screen.getByRole('button', { name: 'Dar de baja' })).toBeInTheDocument()
  })

  it('shows the API error when the baja fails', async () => {
    const user = userEvent.setup()
    listSurveyorsMock.mockResolvedValue([ana])
    deactivateSurveyorMock.mockRejectedValue(new Error('El agrimensor ya está dado de baja'))
    renderSurveyorsPage()

    await screen.findByText('Ana Gómez')
    await user.click(screen.getByRole('button', { name: 'Dar de baja' }))
    await user.click(screen.getByRole('button', { name: 'Confirmar' }))

    expect(await screen.findByText('El agrimensor ya está dado de baja')).toBeInTheDocument()
  })

  it('shows the API error when the list cannot be loaded', async () => {
    listSurveyorsMock.mockRejectedValue(new Error('No tenés permisos para esta acción'))
    renderSurveyorsPage()

    expect(
      await screen.findByText('No tenés permisos para esta acción'),
    ).toBeInTheDocument()
  })

  it('reports an unexpected failure with a generic message', async () => {
    listSurveyorsMock.mockRejectedValue('boom')
    renderSurveyorsPage()

    expect(await screen.findByText('Ocurrió un error inesperado.')).toBeInTheDocument()
  })

  it('refuses the section for a non-administrador role', async () => {
    renderSurveyorsPage('agrimensor')

    expect(
      await screen.findByText('Solo el administrador puede gestionar agrimensores.'),
    ).toBeInTheDocument()
    expect(listSurveyorsMock).not.toHaveBeenCalled()
    expect(screen.queryByRole('button', { name: 'Nuevo agrimensor' })).not.toBeInTheDocument()
  })
})
