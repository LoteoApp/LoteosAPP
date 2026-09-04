import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Link, MemoryRouter, Route, Routes } from 'react-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import LoteoDetailPage from './LoteoDetailPage'
import { ApiError } from '../../../shared/api/client'
import type { UpdateCallePayload } from '../api/update-calle'
import type { UpdateLotePayload } from '../lib/loteFormValues'
import type { LoteoCalle, LoteoDetail, LoteoLote } from '../types'

const getLoteoMock = vi.fn<(loteoId: string, token: string) => Promise<LoteoDetail>>()
const updateLoteMock = vi.fn<
  (loteoId: string, loteId: string, payload: UpdateLotePayload, token: string) => Promise<LoteoLote>
>()
const updateCalleMock = vi.fn<
  (loteoId: string, calleId: string, payload: UpdateCallePayload, token: string) => Promise<LoteoCalle>
>()

vi.mock('../api/get-loteo', () => ({
  getLoteo: (loteoId: string, token: string) => getLoteoMock(loteoId, token),
}))

vi.mock('../api/update-lote', () => ({
  updateLote: (
    loteoId: string,
    loteId: string,
    payload: UpdateLotePayload,
    token: string,
  ) => updateLoteMock(loteoId, loteId, payload, token),
}))

vi.mock('../api/update-calle', () => ({
  updateCalle: (
    loteoId: string,
    calleId: string,
    payload: UpdateCallePayload,
    token: string,
  ) => updateCalleMock(loteoId, calleId, payload, token),
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
      { id: 'mz-1', numero: '1', tieneAgua: false, tieneCloaca: false, tieneLuz: false, tieneGas: false, calleIds: [], poligono: triangle },
      { id: 'mz-2', numero: '2', tieneAgua: false, tieneCloaca: false, tieneLuz: false, tieneGas: false, calleIds: [], poligono: triangle },
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
      <Link to="/lotes/loteo-2">ir a loteo-2</Link>
      <Routes>
        <Route
          path="/lotes/:loteoId"
          element={<LoteoDetailPage accessToken="token-123" canEdit />}
        />
      </Routes>
    </MemoryRouter>,
  )
}

afterEach(() => {
  getLoteoMock.mockReset()
  updateLoteMock.mockReset()
  updateCalleMock.mockReset()
})

describe('LoteoDetailPage', () => {
  it('renders the loteo metadata and its lotes once loaded', async () => {
    getLoteoMock.mockResolvedValue(detail())
    renderPage()

    expect(await screen.findByRole('heading', { name: 'Las Acacias' })).toBeInTheDocument()
    expect(screen.getByText('Río Ceballos, Córdoba')).toBeInTheDocument()

    const rows = screen.getAllByRole('row')
    expect(rows).toHaveLength(3)
    expect(within(rows[1]).getByText(/150\.000/)).toBeInTheDocument()
    expect(within(rows[1]).getByText('300 m²')).toBeInTheDocument()
  })

  it('draws the persisted plan and exposes the layer toggles', async () => {
    getLoteoMock.mockResolvedValue(detail())
    renderPage()

    expect(await screen.findByRole('group', { name: 'Plano del loteo' })).toBeInTheDocument()
    expect(screen.getByRole('group', { name: 'Capas del plano' })).toBeInTheDocument()
  })

  it('filters the lotes table by manzana', async () => {
    getLoteoMock.mockResolvedValue(detail())
    renderPage()

    await screen.findByRole('heading', { name: 'Las Acacias' })
    expect(screen.getAllByRole('row')).toHaveLength(3)

    await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Manzana' }), 'mz-2')

    const rows = screen.getAllByRole('row')
    expect(rows).toHaveLength(2)
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

  it('drops the previous manzana filter when navigating to another loteo', async () => {
    getLoteoMock.mockImplementation(async (loteoId) =>
      loteoId === 'loteo-2'
        ? detail({
            id: 'loteo-2',
            nombre: 'Altos del Sur',
            manzanas: [
              { id: 'mz-9', numero: '9', tieneAgua: false, tieneCloaca: false, tieneLuz: false, tieneGas: false, calleIds: [], poligono: triangle },
              { id: 'mz-10', numero: '10', tieneAgua: false, tieneCloaca: false, tieneLuz: false, tieneGas: false, calleIds: [], poligono: triangle },
            ],
            lotes: [
              { ...detail().lotes[0], id: 'lt-9', manzanaId: 'mz-9', numero: '90' },
              { ...detail().lotes[1], id: 'lt-10', manzanaId: 'mz-10', numero: '91' },
            ],
          })
        : detail(),
    )
    renderPage()

    await screen.findByRole('heading', { name: 'Las Acacias' })
    await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Manzana' }), 'mz-2')
    expect(screen.getAllByRole('row')).toHaveLength(2)

    await userEvent.click(screen.getByRole('link', { name: 'ir a loteo-2' }))

    await screen.findByRole('heading', { name: 'Altos del Sur' })
    expect(screen.getByRole('combobox', { name: 'Manzana' })).toHaveValue('')
    const rows = screen.getAllByRole('row')
    expect(within(rows[1]).getByText('90')).toBeInTheDocument()
    expect(within(rows[2]).getByText('91')).toBeInTheDocument()
  })

  it('tells the user when the loteo has no plan yet', async () => {
    getLoteoMock.mockResolvedValue(
      detail({
        contorno: [],
        manzanas: [{ id: 'mz-1', numero: '1', tieneAgua: false, tieneCloaca: false, tieneLuz: false, tieneGas: false, calleIds: [], poligono: [] }],
        lotes: [],
        calles: [],
      }),
    )
    renderPage()

    expect(
      await screen.findByText('Este loteo todavía no tiene un plano cargado.'),
    ).toBeInTheDocument()
    expect(screen.queryByRole('img', { name: 'Plano del loteo' })).not.toBeInTheDocument()
    expect(screen.queryByRole('group', { name: 'Plano del loteo' })).not.toBeInTheDocument()
  })

  it('edits a lote from the plan and updates the table without reloading', async () => {
    const user = userEvent.setup()
    getLoteoMock.mockResolvedValue(detail())
    updateLoteMock.mockResolvedValue({
      ...detail().lotes[0],
      numero: '12',
      precio: 200000,
      moneda: 'ARS',
      superficie: 310,
      caracteristicas: 'Frente norte',
    })

    renderPage()

    await screen.findByRole('heading', { name: 'Las Acacias' })
    await user.click(screen.getByRole('button', { name: 'Lote 7' }))

    expect(screen.getByLabelText('Número')).toHaveValue('7')
    await user.clear(screen.getByLabelText('Número'))
    await user.type(screen.getByLabelText('Número'), '12')
    await user.click(screen.getByRole('button', { name: 'Guardar' }))

    expect(await screen.findByText('Lote guardado')).toBeInTheDocument()
    expect(within(screen.getAllByRole('row')[1]).getByText('12')).toBeInTheDocument()
  })

  it('selects a lote from the table and shows the form', async () => {
    const user = userEvent.setup()
    getLoteoMock.mockResolvedValue(detail())
    renderPage()

    await screen.findByRole('heading', { name: 'Las Acacias' })
    await user.click(screen.getByRole('row', { name: 'Lote 8' }))

    expect(screen.getByLabelText('Número')).toHaveValue('8')
    expect(screen.getByRole('button', { name: 'Lote 8' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('lets the user edit a manzana and a calle', async () => {
    const user = userEvent.setup()
    getLoteoMock.mockResolvedValue(detail())
    updateCalleMock.mockResolvedValue({
      id: 'ca-1',
      nombre: 'San Martín',
      tipo: 'tierra',
      poligono: triangle,
    })
    renderPage()

    await screen.findByRole('heading', { name: 'Las Acacias' })
    await user.click(screen.getByRole('button', { name: 'Manzana 1' }))
    expect(screen.getByLabelText('Número')).toHaveValue('1')
    expect(screen.getByRole('button', { name: 'Agua' })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Calle Los Álamos' }))
    expect(screen.getByLabelText('Nombre')).toHaveValue('Los Álamos')
    await user.clear(screen.getByLabelText('Nombre'))
    await user.type(screen.getByLabelText('Nombre'), 'San Martín')
    await user.click(screen.getByRole('button', { name: 'Tierra' }))
    await user.click(screen.getByRole('button', { name: 'Guardar' }))

    expect(await screen.findByText('Calle guardada')).toBeInTheDocument()
    expect(updateCalleMock).toHaveBeenCalledWith(
      'loteo-1',
      'ca-1',
      { nombre: 'San Martín', tipo: 'tierra' },
      'token-123',
    )
  })
})
