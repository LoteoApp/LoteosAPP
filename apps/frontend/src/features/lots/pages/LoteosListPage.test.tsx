import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import LoteosListPage from './LoteosListPage'
import type { LoteoSummary } from '../types'
import type { UseLoteos } from '../hooks/use-loteos'

const useLoteosMock = vi.fn<(token: string, search: string) => UseLoteos>()

vi.mock('../hooks/use-loteos', () => ({
  useLoteos: (token: string, search: string) => useLoteosMock(token, search),
}))

function summary(overrides: Partial<LoteoSummary> = {}): LoteoSummary {
  return {
    id: 'loteo-1',
    nombre: 'Loteo Las Acacias',
    ubicacion: 'Río Ceballos, Córdoba',
    descripcion: 'Loteo residencial sobre ruta E-53.',
    cantidadManzanas: 12,
    cantidadLotes: 148,
    cantidadCalles: 8,
    tienePlano: true,
    tieneDxf: true,
    fechaCreacion: '2026-01-10T12:00:00Z',
    ...overrides,
  }
}

function renderPage() {
  render(
    <MemoryRouter>
      <LoteosListPage accessToken="token-123" />
    </MemoryRouter>,
  )
}

afterEach(() => {
  useLoteosMock.mockReset()
})

describe('LoteosListPage', () => {
  it('renders every loteo with its data', () => {
    useLoteosMock.mockReturnValue({
      loteos: [
        summary({ id: 'a', nombre: 'Loteo Las Acacias' }),
        summary({ id: 'b', nombre: 'Altos del Sur', tienePlano: false }),
      ],
      isLoading: false,
      error: null,
    })

    renderPage()

    expect(screen.getByRole('link', { name: /Loteo Las Acacias/ })).toHaveAttribute(
      'href',
      '/lotes/a',
    )
    expect(screen.getByRole('link', { name: /Altos del Sur/ })).toHaveAttribute(
      'href',
      '/lotes/b',
    )
    expect(screen.getByText('Plano cargado')).toBeInTheDocument()
    expect(screen.getByText('Plano pendiente')).toBeInTheDocument()
  })

  it('shows the loading state before the first response', () => {
    useLoteosMock.mockReturnValue({ loteos: [], isLoading: true, error: null })

    renderPage()

    expect(screen.getByText('Cargando loteos…')).toBeInTheDocument()
  })

  it('shows the backend error message', () => {
    useLoteosMock.mockReturnValue({ loteos: [], isLoading: false, error: 'No autorizado' })

    renderPage()

    expect(screen.getByText('No se pudo cargar el listado')).toBeInTheDocument()
    expect(screen.getByText('No autorizado')).toBeInTheDocument()
  })

  it('shows an empty state when there are no loteos yet', () => {
    useLoteosMock.mockReturnValue({ loteos: [], isLoading: false, error: null })

    renderPage()

    expect(screen.getByText('Todavía no hay loteos cargados.')).toBeInTheDocument()
  })

  it('filters the list through the search box', async () => {
    const user = userEvent.setup()
    const all = [
      summary({ id: 'a', nombre: 'Loteo Las Acacias' }),
      summary({ id: 'b', nombre: 'Altos del Sur' }),
    ]
    useLoteosMock.mockImplementation((_token, search) => ({
      loteos: search
        ? all.filter((loteo) => loteo.nombre.toLowerCase().includes(search.toLowerCase()))
        : all,
      isLoading: false,
      error: null,
    }))

    renderPage()

    expect(screen.getByRole('link', { name: /Altos del Sur/ })).toBeInTheDocument()

    await user.type(screen.getByLabelText('Buscar'), 'acacias')

    expect(screen.queryByRole('link', { name: /Altos del Sur/ })).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Loteo Las Acacias/ })).toBeInTheDocument()
  })

  it('reports when a search matches nothing', async () => {
    const user = userEvent.setup()
    useLoteosMock.mockImplementation((_token, search) => ({
      loteos: search ? [] : [summary()],
      isLoading: false,
      error: null,
    }))

    renderPage()

    await user.type(screen.getByLabelText('Buscar'), 'zzz')

    expect(screen.getByText('No se encontraron loteos con esa búsqueda.')).toBeInTheDocument()
  })

  it('links to the alta form', () => {
    useLoteosMock.mockReturnValue({ loteos: [], isLoading: false, error: null })

    renderPage()

    expect(screen.getByRole('link', { name: 'Nuevo loteo' })).toHaveAttribute(
      'href',
      '/lotes/nuevo',
    )
  })
})
