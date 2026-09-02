import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import LoteoDetailPage from './LoteoDetailPage'
import { ApiError } from '../../../shared/api/client'
import type { LoteoDetail } from '../types'

const getLoteoMock = vi.fn<(loteoId: string, token: string) => Promise<LoteoDetail>>()

vi.mock('../api/get-loteo', () => ({
  getLoteo: (loteoId: string, token: string) => getLoteoMock(loteoId, token),
}))

const triangle = [
  { x: 0, y: 0 },
  { x: 10, y: 0 },
  { x: 10, y: 10 },
]

function detail(overrides: Partial<LoteoDetail> = {}): LoteoDetail {
  return {
    id: 'loteo-1',
    nombre: 'Las Acacias',
    ubicacion: 'Río Ceballos, Córdoba',
    descripcion: 'Sobre ruta E-53.',
    contorno: triangle,
    manzanas: [
      { id: 'mz-1', numero: '1', poligono: triangle },
      { id: 'mz-2', numero: '2', poligono: triangle },
    ],
    lotes: [
      {
        id: 'lt-1',
        manzanaId: 'mz-1',
        numero: '7',
        precio: 150000,
        moneda: 'USD',
        superficie: 300,
        caracteristicas: '',
        poligono: triangle,
      },
      {
        id: 'lt-2',
        manzanaId: 'mz-2',
        numero: '8',
        precio: 90000,
        moneda: 'USD',
        superficie: 250,
        caracteristicas: '',
        poligono: triangle,
      },
    ],
    calles: [{ id: 'ca-1', nombre: 'Los Álamos', tipo: 'asfalto', poligono: triangle }],
    fechaCreacion: '2026-08-20T12:00:00Z',
    ...overrides,
  }
}

function renderPage(path = '/lotes/loteo-1') {
  render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route
          path="/lotes/:loteoId"
          element={<LoteoDetailPage accessToken="token-123" />}
        />
      </Routes>
    </MemoryRouter>,
  )
}

afterEach(() => {
  getLoteoMock.mockReset()
})

describe('LoteoDetailPage', () => {
  it('renders the loteo metadata and its lotes once loaded', async () => {
    getLoteoMock.mockResolvedValue(detail())
    renderPage()

    expect(await screen.findByRole('heading', { name: 'Las Acacias' })).toBeInTheDocument()
    expect(screen.getByText('Río Ceballos, Córdoba')).toBeInTheDocument()

    const rows = screen.getAllByRole('row')
    expect(rows).toHaveLength(3) // header + 2 lotes
    expect(within(rows[1]).getByText(/150\.000/)).toBeInTheDocument()
    expect(within(rows[1]).getByText('300 m²')).toBeInTheDocument()
  })

  it('draws the persisted plan and exposes the layer toggles', async () => {
    getLoteoMock.mockResolvedValue(detail())
    renderPage()

    expect(await screen.findByRole('img', { name: 'Plano del loteo' })).toBeInTheDocument()
    expect(screen.getByRole('group', { name: 'Capas del plano' })).toBeInTheDocument()
  })

  it('filters the lotes table by manzana', async () => {
    getLoteoMock.mockResolvedValue(detail())
    renderPage()

    await screen.findByRole('heading', { name: 'Las Acacias' })
    expect(screen.getAllByRole('row')).toHaveLength(3)

    await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Manzana' }), 'mz-2')

    const rows = screen.getAllByRole('row')
    expect(rows).toHaveLength(2) // header + lote 8
    expect(within(rows[1]).getByText('8')).toBeInTheDocument()
  })

  it('shows a not-found panel when the loteo does not exist', async () => {
    getLoteoMock.mockRejectedValue(
      new ApiError('El loteo solicitado no existe', 'loteo_not_found', 404),
    )
    renderPage('/lotes/missing')

    expect(await screen.findByText('No encontramos este loteo')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Volver al listado' })).toHaveAttribute(
      'href',
      '/lotes',
    )
  })

  it('shows an error alert when the request fails', async () => {
    getLoteoMock.mockRejectedValue(new Error('No se pudo cargar el loteo, intentá nuevamente.'))
    renderPage()

    expect(await screen.findByText('No se pudo cargar el loteo')).toBeInTheDocument()
    expect(screen.getByText('No se pudo cargar el loteo, intentá nuevamente.')).toBeInTheDocument()
  })

  it('tells the user when the loteo has no plan yet', async () => {
    getLoteoMock.mockResolvedValue(
      detail({
        contorno: [],
        manzanas: [{ id: 'mz-1', numero: '1', poligono: [] }],
        lotes: [],
        calles: [],
      }),
    )
    renderPage()

    expect(
      await screen.findByText('Este loteo todavía no tiene un plano cargado.'),
    ).toBeInTheDocument()
    expect(screen.queryByRole('img', { name: 'Plano del loteo' })).not.toBeInTheDocument()
  })
})
