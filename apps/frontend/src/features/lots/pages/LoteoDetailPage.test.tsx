import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Link, MemoryRouter, Route, Routes } from 'react-router'
import type { ComponentProps } from 'react'
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
        estado: 'disponible',
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
        estado: 'reservado',
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

function renderPage(
  path = '/lotes/loteo-1',
  renderReservationAction?: ComponentProps<typeof LoteoDetailPage>['renderReservationAction'],
  renderReservationSummary?: ComponentProps<typeof LoteoDetailPage>['renderReservationSummary'],
  renderReservations?: ComponentProps<typeof LoteoDetailPage>['renderReservations'],
) {
  render(
    <MemoryRouter initialEntries={[path]}>
      <Link to="/lotes/loteo-2">ir a loteo-2</Link>
      <Routes>
        <Route
          path="/lotes/:loteoId"
          element={
            <LoteoDetailPage
              accessToken="token-123"
              canEdit
              renderReservationAction={renderReservationAction}
              renderReservationSummary={renderReservationSummary}
              renderReservations={renderReservations}
            />
          }
        />
      </Routes>
    </MemoryRouter>,
  )
}

function lotesRows() {
  return within(screen.getByRole('list', { name: 'Lotes del loteo' })).getAllByRole('listitem')
}

function planButton(name: string) {
  return within(screen.getByRole('group', { name: 'Plano del loteo' })).getByRole('button', { name })
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

    await userEvent.click(screen.getByRole('tab', { name: 'Lotes' }))

    const rows = lotesRows()
    expect(rows).toHaveLength(2)
    expect(within(rows[0]).getByText(/150\.000/)).toBeInTheDocument()
    expect(within(rows[0]).getByText('300 m²')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Lote 7 · Mz 1 · Disponible' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Lote 8 · Mz 2 · Reservado' })).toBeInTheDocument()
  })

  it('draws the persisted plan and exposes the layer toggles', async () => {
    getLoteoMock.mockResolvedValue(detail())
    renderPage()

    expect(await screen.findByRole('group', { name: 'Plano del loteo' })).toBeInTheDocument()
    expect(screen.getByRole('group', { name: 'Capas del plano' })).toBeInTheDocument()
  })

  it('marks the selected lot as reserved in the plan after the reservation action completes', async () => {
    const user = userEvent.setup()
    const renderReservationAction = vi.fn(
      (_lote: LoteoLote, onCreated: () => void) => (
        <button type="button" onClick={onCreated}>Confirmar desde visor</button>
      ),
    )
    getLoteoMock.mockResolvedValue(detail())
    renderPage('/lotes/loteo-1', renderReservationAction)

    await screen.findByRole('heading', { name: 'Las Acacias' })
    await user.click(screen.getByRole('button', { name: 'Lote 7' }))
    await user.click(screen.getByRole('button', { name: 'Confirmar desde visor' }))

    expect(await screen.findByText('Reserva creada')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Lote 7 · Mz 1 · Reservado' })).toBeInTheDocument()
    expect(
      screen
        .getByRole('group', { name: 'Plano del loteo' })
        .querySelector('[aria-label="Lote 7"]'),
    ).toHaveAttribute('fill', 'var(--lot-reserved-plan)')
    expect(renderReservationAction).toHaveBeenCalledWith(
      expect.objectContaining({ id: 'lt-1', estado: 'disponible' }),
      expect.any(Function),
    )
  })

  it('marks the selected lot as available in the plan after cancellation', async () => {
    const user = userEvent.setup()
    const renderReservationSummary = vi.fn(
      (_lote: LoteoLote, onCanceled: () => void) => (
        <button type="button" onClick={onCanceled}>Cancelar desde visor</button>
      ),
    )
    getLoteoMock.mockResolvedValue(detail())
    renderPage('/lotes/loteo-1', undefined, renderReservationSummary)

    await screen.findByRole('heading', { name: 'Las Acacias' })
    await user.click(planButton('Lote 8'))
    await user.click(screen.getByRole('button', { name: 'Cancelar desde visor' }))

    expect(screen.getByText('Reserva cancelada')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Lote 8 · Mz 2 · Disponible' })).toBeInTheDocument()
    expect(
      screen
        .getByRole('group', { name: 'Plano del loteo' })
        .querySelector('[aria-label="Lote 8"]'),
    ).toHaveAttribute('fill', 'var(--lot-available-plan)')
    expect(renderReservationSummary).toHaveBeenCalledWith(
      expect.objectContaining({ id: 'lt-2', estado: 'reservado' }),
      expect.any(Function),
    )
  })

  it('filters the lotes with the search box', async () => {
    const user = userEvent.setup()
    getLoteoMock.mockResolvedValue(detail())
    renderPage()

    await screen.findByRole('heading', { name: 'Las Acacias' })
    await user.click(screen.getByRole('tab', { name: 'Lotes' }))
    expect(lotesRows()).toHaveLength(2)

    await user.type(screen.getByRole('searchbox', { name: 'Buscar lote o manzana' }), '8')

    expect(lotesRows()).toHaveLength(1)
    expect(screen.getByRole('button', { name: 'Lote 8 · Mz 2 · Reservado' })).toBeInTheDocument()
  })

  it('filters by the exact manzana id even when lot and manzana numbers overlap', async () => {
    const user = userEvent.setup()
    getLoteoMock.mockResolvedValue(
      detail({
        manzanas: [
          { ...detail().manzanas[0], id: 'mz-1', numero: '1' },
          { ...detail().manzanas[1], id: 'mz-10', numero: '10' },
        ],
        lotes: [
          { ...detail().lotes[0], id: 'lt-7', manzanaId: 'mz-1', numero: '7' },
          { ...detail().lotes[1], id: 'lt-1', manzanaId: 'mz-10', numero: '1' },
        ],
      }),
    )
    renderPage()

    await screen.findByRole('heading', { name: 'Las Acacias' })
    await user.click(screen.getByRole('tab', { name: 'Lotes' }))
    await user.selectOptions(screen.getByRole('combobox', { name: 'Manzana' }), 'mz-1')

    expect(lotesRows()).toHaveLength(1)
    expect(screen.getByRole('button', { name: 'Lote 7 · Mz 1 · Disponible' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Lote 1 · Mz 10 · Reservado' })).not.toBeInTheDocument()
  })

  it('filters the lotes by state', async () => {
    const user = userEvent.setup()
    getLoteoMock.mockResolvedValue(detail())
    renderPage()

    await screen.findByRole('heading', { name: 'Las Acacias' })
    await user.click(screen.getByRole('tab', { name: 'Lotes' }))

    const filters = within(screen.getByRole('group', { name: 'Filtrar por estado' }))
    await user.click(filters.getByRole('button', { name: 'Reservado' }))

    expect(lotesRows()).toHaveLength(1)
    expect(screen.getByRole('button', { name: 'Lote 8 · Mz 2 · Reservado' })).toBeInTheDocument()
  })

  it('opens on the summary tab with the lotes counted by state', async () => {
    getLoteoMock.mockResolvedValue(detail())
    renderPage()

    await screen.findByRole('heading', { name: 'Las Acacias' })

    expect(screen.getByRole('tab', { name: 'Resumen' })).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByLabelText('1 Disponible')).toBeInTheDocument()
    expect(screen.getByLabelText('1 Reservado')).toBeInTheDocument()
    expect(screen.queryByRole('list', { name: 'Lotes del loteo' })).not.toBeInTheDocument()
  })

  it('jumps to the lotes tab when the user picks something on the plan', async () => {
    const user = userEvent.setup()
    getLoteoMock.mockResolvedValue(detail())
    renderPage()

    await screen.findByRole('heading', { name: 'Las Acacias' })
    await user.click(screen.getByRole('button', { name: 'Lote 7' }))

    expect(screen.getByRole('tab', { name: 'Lotes' })).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByRole('list', { name: 'Lotes del loteo' })).toBeInTheDocument()
  })

  it('goes back to the summary tab when navigating to another loteo', async () => {
    const user = userEvent.setup()
    getLoteoMock.mockImplementation(async (loteoId) =>
      loteoId === 'loteo-2' ? detail({ id: 'loteo-2', nombre: 'Altos del Sur' }) : detail(),
    )
    renderPage()

    await screen.findByRole('heading', { name: 'Las Acacias' })
    await user.click(screen.getByRole('tab', { name: 'Lotes' }))
    expect(screen.getByRole('tab', { name: 'Lotes' })).toHaveAttribute('aria-selected', 'true')

    await user.click(screen.getByRole('link', { name: 'ir a loteo-2' }))

    await screen.findByRole('heading', { name: 'Altos del Sur' })
    expect(screen.getByRole('tab', { name: 'Resumen' })).toHaveAttribute('aria-selected', 'true')
  })

  it('hides the reservations tab when the user cannot see reservations', async () => {
    getLoteoMock.mockResolvedValue(detail())
    renderPage()

    await screen.findByRole('heading', { name: 'Las Acacias' })

    expect(screen.queryByRole('tab', { name: 'Reservas' })).not.toBeInTheDocument()
  })

  it('frees the lote in the plan when a reservation is canceled from the reservations tab', async () => {
    const user = userEvent.setup()
    getLoteoMock.mockResolvedValue(detail())
    renderPage('/lotes/loteo-1', undefined, undefined, (loteo, onReservationCanceled) => (
      <button type="button" onClick={() => onReservationCanceled(loteo.lotes[1].id)}>
        Cancelar reserva del lote 8
      </button>
    ))

    await screen.findByRole('heading', { name: 'Las Acacias' })
    await user.click(screen.getByRole('tab', { name: 'Reservas' }))
    await user.click(screen.getByRole('button', { name: 'Cancelar reserva del lote 8' }))

    expect(screen.getByText('Reserva cancelada')).toBeInTheDocument()
    expect(
      screen
        .getByRole('group', { name: 'Plano del loteo' })
        .querySelector('[aria-label="Lote 8"]'),
    ).toHaveAttribute('fill', 'var(--lot-available-plan)')
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

  it('drops the previous search when navigating to another loteo', async () => {
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
    await userEvent.click(screen.getByRole('tab', { name: 'Lotes' }))
    await userEvent.type(screen.getByRole('searchbox', { name: 'Buscar lote o manzana' }), '8')
    expect(lotesRows()).toHaveLength(1)

    await userEvent.click(screen.getByRole('link', { name: 'ir a loteo-2' }))

    await screen.findByRole('heading', { name: 'Altos del Sur' })
    await userEvent.click(screen.getByRole('tab', { name: 'Lotes' }))
    expect(screen.getByRole('searchbox', { name: 'Buscar lote o manzana' })).toHaveValue('')
    expect(lotesRows()).toHaveLength(2)
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
    await user.click(screen.getByRole('button', { name: 'Habilitar edición' }))

    expect(screen.getByLabelText('Número')).toHaveValue('7')
    await user.clear(screen.getByLabelText('Número'))
    await user.type(screen.getByLabelText('Número'), '12')
    await user.click(screen.getByRole('button', { name: 'Guardar' }))

    expect(await screen.findByText('Lote guardado')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Lote 12 · Mz 1 · Disponible' })).toBeInTheDocument()
  })

  it('selects a lote from the list and shows its data without opening edit mode', async () => {
    const user = userEvent.setup()
    getLoteoMock.mockResolvedValue(detail())
    renderPage()

    await screen.findByRole('heading', { name: 'Las Acacias' })
    await user.click(screen.getByRole('tab', { name: 'Lotes' }))
    await user.click(screen.getByRole('button', { name: 'Lote 8 · Mz 2 · Reservado' }))

    expect(screen.getByText('Manzana 2 · 250 m²')).toBeInTheDocument()
    expect(screen.queryByLabelText('Número')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Habilitar edición' })).toBeInTheDocument()
    expect(planButton('Lote 8')).toHaveAttribute('aria-pressed', 'true')
  })

  it('starts a newly selected lote in read-only mode after another lote was edited', async () => {
    const user = userEvent.setup()
    getLoteoMock.mockResolvedValue(detail())
    renderPage()

    await screen.findByRole('heading', { name: 'Las Acacias' })
    await user.click(screen.getByRole('button', { name: 'Lote 7' }))
    await user.click(screen.getByRole('button', { name: 'Habilitar edición' }))
    expect(screen.getByLabelText('Número')).toHaveValue('7')

    await user.click(planButton('Lote 8'))

    expect(screen.queryByLabelText('Número')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Habilitar edición' })).toBeInTheDocument()
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

  it('clears a stale save error banner when the selection changes', async () => {
    const user = userEvent.setup()
    getLoteoMock.mockResolvedValue(detail())
    updateLoteMock.mockRejectedValueOnce(
      new ApiError('Ocurrió un error inesperado.', 'server_error', 500),
    )

    renderPage()

    await screen.findByRole('heading', { name: 'Las Acacias' })
    await user.click(screen.getByRole('button', { name: 'Lote 7' }))
    await user.click(screen.getByRole('button', { name: 'Habilitar edición' }))
    await user.click(screen.getByRole('button', { name: 'Guardar' }))

    expect(await screen.findByText('Ocurrió un error inesperado.')).toBeInTheDocument()

    await user.click(planButton('Lote 8'))

    expect(screen.queryByText('Ocurrió un error inesperado.')).not.toBeInTheDocument()
  })
})
