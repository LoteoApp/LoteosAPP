import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import LoteosListPage from './LoteosListPage'
import type { LoteoSummary } from '../types'

const listLoteosMock = vi.fn<(token: string, options?: { q?: string }) => Promise<LoteoSummary[]>>()

vi.mock('../api/list-loteos', () => ({
  listLoteos: (token: string, options?: { q?: string }) => listLoteosMock(token, options),
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
  listLoteosMock.mockReset()
})

describe('LoteosListPage', () => {
  it('renders every loteo with its data', async () => {
    listLoteosMock.mockResolvedValue([
      summary({ id: 'a', nombre: 'Loteo Las Acacias' }),
      summary({ id: 'b', nombre: 'Altos del Sur', tienePlano: false }),
    ])

    renderPage()

    expect(await screen.findByRole('link', { name: /Loteo Las Acacias/ })).toHaveAttribute(
      'href',
      '/lotes/a',
    )
    expect(screen.getByRole('link', { name: /Altos del Sur/ })).toHaveAttribute('href', '/lotes/b')
    expect(screen.getByText('Plano cargado')).toBeInTheDocument()
    expect(screen.getByText('Plano pendiente')).toBeInTheDocument()
  })

  it('shows the loading state before the first response', () => {
    listLoteosMock.mockReturnValue(new Promise(() => {}))

    renderPage()

    expect(screen.getByText('Cargando loteos…')).toBeInTheDocument()
  })

  it('shows the backend error message', async () => {
    listLoteosMock.mockRejectedValue(new Error('No autorizado'))

    renderPage()

    expect(await screen.findByText('No se pudo cargar el listado')).toBeInTheDocument()
    expect(screen.getByText('No autorizado')).toBeInTheDocument()
  })

  it('drops the previous list when a refresh fails', async () => {
    const user = userEvent.setup()
    listLoteosMock.mockResolvedValueOnce([summary({ nombre: 'Loteo Las Acacias' })])

    renderPage()

    expect(await screen.findByRole('link', { name: /Loteo Las Acacias/ })).toBeInTheDocument()

    listLoteosMock.mockRejectedValueOnce(new Error('No autorizado'))
    await user.type(screen.getByLabelText('Buscar'), 'acacias')

    expect(await screen.findByText('No autorizado')).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /Loteo Las Acacias/ })).not.toBeInTheDocument()
  })

  it('shows an empty state when there are no loteos yet', async () => {
    listLoteosMock.mockResolvedValue([])

    renderPage()

    expect(await screen.findByText('Todavía no hay loteos cargados.')).toBeInTheDocument()
  })

  it('filters the list through the search box', async () => {
    const user = userEvent.setup()
    const all = [
      summary({ id: 'a', nombre: 'Loteo Las Acacias' }),
      summary({ id: 'b', nombre: 'Altos del Sur' }),
    ]
    listLoteosMock.mockImplementation((_token, options) =>
      Promise.resolve(
        options?.q
          ? all.filter((loteo) => loteo.nombre.toLowerCase().includes(options.q!.toLowerCase()))
          : all,
      ),
    )

    renderPage()

    expect(await screen.findByRole('link', { name: /Altos del Sur/ })).toBeInTheDocument()

    await user.type(screen.getByLabelText('Buscar'), 'acacias')

    await waitFor(() =>
      expect(screen.queryByRole('link', { name: /Altos del Sur/ })).not.toBeInTheDocument(),
    )
    expect(screen.getByRole('link', { name: /Loteo Las Acacias/ })).toBeInTheDocument()
  })

  it('reports when a search matches nothing', async () => {
    const user = userEvent.setup()
    listLoteosMock.mockImplementation((_token, options) =>
      Promise.resolve(options?.q ? [] : [summary()]),
    )

    renderPage()

    expect(await screen.findByRole('link', { name: /Loteo Las Acacias/ })).toBeInTheDocument()

    await user.type(screen.getByLabelText('Buscar'), 'zzz')

    expect(
      await screen.findByText('No se encontraron loteos con esa búsqueda.'),
    ).toBeInTheDocument()
  })

  it('keeps the search box mounted while a cleared search refetches', async () => {
    const user = userEvent.setup()
    listLoteosMock.mockImplementation((_token, options) =>
      Promise.resolve(options?.q ? [] : [summary()]),
    )

    renderPage()

    expect(await screen.findByRole('link', { name: /Loteo Las Acacias/ })).toBeInTheDocument()

    const searchBox = screen.getByLabelText('Buscar')
    await user.type(searchBox, 'zzz')
    expect(
      await screen.findByText('No se encontraron loteos con esa búsqueda.'),
    ).toBeInTheDocument()

    await user.clear(searchBox)

    expect(screen.getByLabelText('Buscar')).toBe(searchBox)
    expect(searchBox).toHaveFocus()
    expect(await screen.findByRole('link', { name: /Loteo Las Acacias/ })).toBeInTheDocument()
  })

  it('links to the alta form', async () => {
    listLoteosMock.mockResolvedValue([])

    renderPage()

    expect(await screen.findByRole('link', { name: 'Nuevo loteo' })).toHaveAttribute(
      'href',
      '/lotes/nuevo',
    )
  })
})
