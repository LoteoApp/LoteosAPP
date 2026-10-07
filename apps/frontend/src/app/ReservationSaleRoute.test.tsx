import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Link, MemoryRouter, Route, Routes } from 'react-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '../shared/api/client'
import ReservationSaleRoute from './ReservationSaleRoute'
import type { LoteoDetail } from '../features/lots/types'
import type { Reservation } from '../features/reservations/types'

const useLoteoMock = vi.hoisted(() => vi.fn())
const getReservationMock = vi.hoisted(() => vi.fn())
const convertMock = vi.hoisted(() => vi.fn())
const auth = vi.hoisted(() => ({ userId: 'user-1', token: 'token' }))

vi.mock('../features/auth/hooks/use-auth', () => ({
  useAuth: () => ({
    session: { access_token: auth.token },
    user: { id: auth.userId, app_metadata: { role: 'inmobiliaria' } },
  }),
}))
vi.mock('../features/lots/hooks/use-loteo', () => ({ useLoteo: useLoteoMock }))
vi.mock('../features/reservations/api/reservations', () => ({ getReservation: getReservationMock }))
vi.mock('../features/sales/api/sales', () => ({ convertReservationToSale: convertMock }))

const triangle = [{ x: 0, y: 0 }, { x: 10, y: 0 }, { x: 10, y: 10 }]
const development: LoteoDetail = {
  id: 'loteo-1', nombre: 'Las Acacias', ubicacion: 'Córdoba', descripcion: '',
  contorno: triangle,
  manzanas: [{ id: 'block-1', numero: '2', tieneAgua: true, tieneCloaca: false, tieneLuz: true, tieneGas: false, calleIds: [], poligono: triangle }],
  lotes: [
    { id: 'lot-1', manzanaId: 'block-1', numero: '7', estado: 'reservado', precio: 120000, moneda: 'USD', superficie: 300, caracteristicas: '', poligono: triangle },
  ],
  calles: [], fechaCreacion: '2026-08-20T12:00:00Z',
}

function reservation(overrides: Partial<Reservation> = {}): Reservation {
  return {
    id: 'reservation-1', loteoId: 'loteo-1', loteoNombre: 'Las Acacias', loteId: 'lot-1', loteNumero: '7',
    cliente: { id: 'client-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' },
    vendedor: { id: 'seller-1', nombre: 'Beto', apellido: 'Gómez', rol: 'inmobiliaria' },
    usuarioAlta: { id: 'seller-1', nombre: 'Beto', apellido: 'Gómez', rol: 'inmobiliaria' },
    inmobiliaria: { id: 'ag-1', razonSocial: 'Inmobiliaria Sur' },
    estado: 'activa', fechaVencimiento: '2026-10-10T15:00:00Z', fechaCreacion: '2026-09-25T15:00:00Z',
    fechaModificacion: '2026-09-25T15:00:00Z', puedeConvertir: true,
    ...overrides,
  }
}

function renderRoute() {
  return render(
    <MemoryRouter initialEntries={['/reservas/reservation-1/convertir']}>
      <Routes>
        <Route path="/reservas/:id/convertir" element={<ReservationSaleRoute />} />
      </Routes>
    </MemoryRouter>,
  )
}

afterEach(() => {
  vi.clearAllMocks()
  auth.userId = 'user-1'
  auth.token = 'token'
})

function deferredSale() {
  let resolve: (value: unknown) => void = () => undefined
  convertMock.mockImplementation(() => new Promise((done) => { resolve = done }))
  return (value: unknown) => resolve(value)
}

const persistedSale = {
  id: 'sale-1', loteoId: 'loteo-1', loteoNombre: 'Las Acacias', loteId: 'lot-1', loteNumero: '7', manzanaNumero: '2', loteSuperficie: 300,
  cliente: { id: 'client-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' },
  vendedor: { id: 'seller-1', nombre: 'Beto', apellido: 'Gómez', rol: 'inmobiliaria' },
  usuarioAlta: { id: 'seller-1', nombre: 'Beto', apellido: 'Gómez', rol: 'inmobiliaria' },
  modalidadPago: 'contado', monto: 120000, moneda: 'USD', estado: 'completada',
  fechaCreacion: '2026-10-02T15:00:00Z', fechaModificacion: '2026-10-02T15:00:00Z', reservaId: 'reservation-1',
}

describe('ReservationSaleRoute', () => {
  it('composes the reserva and the current lote into the conversion and posts it for this reserva', async () => {
    getReservationMock.mockResolvedValue(reservation())
    useLoteoMock.mockImplementation((loteoId: string) => (loteoId === 'loteo-1' ? { status: 'loaded', loteo: development } : { status: 'loading' }))
    convertMock.mockResolvedValue({
      id: 'sale-1', loteoId: 'loteo-1', loteoNombre: 'Las Acacias', loteId: 'lot-1', loteNumero: '7', manzanaNumero: '2', loteSuperficie: 300,
      cliente: { id: 'client-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' },
      vendedor: { id: 'seller-1', nombre: 'Beto', apellido: 'Gómez', rol: 'inmobiliaria' },
      usuarioAlta: { id: 'seller-1', nombre: 'Beto', apellido: 'Gómez', rol: 'inmobiliaria' },
      modalidadPago: 'contado', monto: 120000, moneda: 'USD', estado: 'completada',
      fechaCreacion: '2026-10-02T15:00:00Z', fechaModificacion: '2026-10-02T15:00:00Z', reservaId: 'reservation-1',
    })
    vi.spyOn(window, 'print').mockImplementation(() => {})
    const user = userEvent.setup()

    renderRoute()

    const fixed = await screen.findByLabelText('Datos de la reserva')
    expect(within(fixed).getByText('Las Acacias · Mz 2 · Lote 7')).toBeInTheDocument()
    expect(within(fixed).getByText('Gómez, Beto · Inmobiliaria Sur')).toBeInTheDocument()
    expect(screen.getByText('US$ 120.000,00')).toBeInTheDocument()
    expect(getReservationMock).toHaveBeenCalledWith('token', 'reservation-1', expect.any(AbortSignal))
    expect(within(screen.getByRole('img', { name: 'Plano del loteo' })).getByLabelText('Lote 7')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))

    expect(convertMock).toHaveBeenCalledWith('token', 'reservation-1', { modalidadPago: 'contado' }, expect.any(String))
    expect(await screen.findByText('Venta registrada')).toBeInTheDocument()
  })

  it('reloads the reserva on request without losing the typed terms', async () => {
    getReservationMock.mockResolvedValue(reservation())
    useLoteoMock.mockReturnValue({ status: 'loaded', loteo: development })
    convertMock.mockRejectedValue(new ApiError('El lote no está disponible para vender', 'sale_lot_unavailable', 409))
    const user = userEvent.setup()

    renderRoute()

    await user.click(await screen.findByLabelText('Condiciones de pago'))
    await user.click(await screen.findByRole('option', { name: 'Financiado' }))
    await user.type(screen.getByLabelText('Cantidad de cuotas'), '24')
    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))
    await user.click(await screen.findByRole('button', { name: 'Actualizar reserva' }))

    await waitFor(() => expect(getReservationMock).toHaveBeenCalledTimes(2))
    expect(screen.getByLabelText('Cantidad de cuotas')).toHaveValue('24')
  })

  it('reports a missing reserva, a failing loteo and a lote no longer in its loteo', async () => {
    getReservationMock.mockRejectedValueOnce(new ApiError('La reserva solicitada no existe', 'reservation_not_found', 404))
    useLoteoMock.mockReturnValue({ status: 'loading' })
    const missing = renderRoute()
    expect(await screen.findByText('No encontramos esta reserva.')).toBeInTheDocument()
    missing.unmount()

    getReservationMock.mockRejectedValueOnce(new Error('Sin conexión'))
    const failing = renderRoute()
    expect(await screen.findByText('Sin conexión')).toBeInTheDocument()
    failing.unmount()

    getReservationMock.mockResolvedValueOnce(reservation())
    useLoteoMock.mockReturnValue({ status: 'error', message: 'boom' })
    const noLoteo = renderRoute()
    expect(await screen.findByText('No se pudo cargar el loteo de la reserva.')).toBeInTheDocument()
    noLoteo.unmount()

    getReservationMock.mockResolvedValueOnce(reservation({ loteId: 'lot-9' }))
    useLoteoMock.mockReturnValue({ status: 'loaded', loteo: development })
    renderRoute()
    expect(await screen.findByText('El lote de la reserva ya no figura en el loteo.')).toBeInTheDocument()
  })

  it('starts over for another user and drops the answer the previous one was waiting for', async () => {
    getReservationMock.mockResolvedValue(reservation())
    useLoteoMock.mockReturnValue({ status: 'loaded', loteo: development })
    const resolve = deferredSale()
    const user = userEvent.setup()
    const view = renderRoute()

    await user.click(await screen.findByRole('button', { name: 'Confirmar venta' }))
    expect(screen.getByRole('button', { name: 'Registrando venta…' })).toBeDisabled()

    auth.userId = 'user-2'
    auth.token = 'token-2'
    view.rerender(
      <MemoryRouter initialEntries={['/reservas/reservation-1/convertir']}>
        <Routes>
          <Route path="/reservas/:id/convertir" element={<ReservationSaleRoute />} />
        </Routes>
      </MemoryRouter>,
    )
    expect(await screen.findByRole('button', { name: 'Confirmar venta' })).toBeEnabled()
    resolve(persistedSale)

    await waitFor(() => expect(screen.getByRole('button', { name: 'Confirmar venta' })).toBeEnabled())
    expect(screen.queryByText('Venta registrada')).not.toBeInTheDocument()
  })

  it('keeps the draft and the attempt across a token refresh of the same user', async () => {
    getReservationMock.mockResolvedValue(reservation())
    useLoteoMock.mockReturnValue({ status: 'loaded', loteo: development })
    const resolve = deferredSale()
    const user = userEvent.setup()
    const view = renderRoute()

    await user.click(await screen.findByLabelText('Condiciones de pago'))
    await user.click(await screen.findByRole('option', { name: 'Financiado' }))
    await user.type(screen.getByLabelText('Cantidad de cuotas'), '12')
    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))
    auth.token = 'refreshed-token'
    view.rerender(
      <MemoryRouter initialEntries={['/reservas/reservation-1/convertir']}>
        <Routes>
          <Route path="/reservas/:id/convertir" element={<ReservationSaleRoute />} />
        </Routes>
      </MemoryRouter>,
    )

    await waitFor(() => expect(getReservationMock).toHaveBeenCalledWith('refreshed-token', 'reservation-1', expect.any(AbortSignal)))
    expect(screen.getByLabelText('Cantidad de cuotas')).toHaveValue('12')
    resolve({ ...persistedSale, estado: 'activa', modalidadPago: 'financiado' })
    expect(await screen.findByText('Venta registrada')).toBeInTheDocument()
    expect(convertMock).toHaveBeenCalledTimes(1)
  })

  it('keeps the form and the uncertain attempt when a reload fails', async () => {
    getReservationMock.mockResolvedValueOnce(reservation()).mockRejectedValue(new Error('Sin conexión.'))
    useLoteoMock.mockReturnValue({ status: 'loaded', loteo: development })
    convertMock
      .mockRejectedValueOnce(new ApiError('No hay conexión', 'network_error', 0))
      .mockResolvedValueOnce({ ...persistedSale, estado: 'activa', modalidadPago: 'financiado' })
    const user = userEvent.setup()
    const view = renderRoute()

    await user.click(await screen.findByLabelText('Condiciones de pago'))
    await user.click(await screen.findByRole('option', { name: 'Financiado' }))
    await user.type(screen.getByLabelText('Cantidad de cuotas'), '12')
    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))
    await screen.findByText(/No pudimos confirmar si la venta se registró/)

    auth.token = 'refreshed-token'
    useLoteoMock.mockReturnValue({ status: 'error', message: 'Sin conexión.' })
    view.rerender(
      <MemoryRouter initialEntries={['/reservas/reservation-1/convertir']}>
        <Routes>
          <Route path="/reservas/:id/convertir" element={<ReservationSaleRoute />} />
        </Routes>
      </MemoryRouter>,
    )

    expect(await screen.findByText('No se pudo actualizar la reserva')).toBeInTheDocument()
    expect(screen.getByLabelText('Cantidad de cuotas')).toHaveValue('12')
    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))

    expect(await screen.findByText('Venta registrada')).toBeInTheDocument()
    expect(convertMock.mock.calls[1][2]).toEqual(convertMock.mock.calls[0][2])
    expect(convertMock.mock.calls[1][3]).toBe(convertMock.mock.calls[0][3])
  })

  it('opens another reserva unblocked while a conversion is still pending', async () => {
    getReservationMock.mockImplementation((_token: string, id: string) => Promise.resolve(reservation({ id, loteoNombre: id })))
    useLoteoMock.mockReturnValue({ status: 'loaded', loteo: development })
    const resolve = deferredSale()
    const user = userEvent.setup()
    render(
      <MemoryRouter initialEntries={['/reservas/reservation-1/convertir']}>
        <Link to="/reservas/reservation-2/convertir">Otra reserva</Link>
        <Routes>
          <Route path="/reservas/:id/convertir" element={<ReservationSaleRoute />} />
        </Routes>
      </MemoryRouter>,
    )

    await user.click(await screen.findByRole('button', { name: 'Confirmar venta' }))
    await user.click(screen.getByRole('link', { name: 'Otra reserva' }))

    await waitFor(() => expect(getReservationMock).toHaveBeenCalledWith('token', 'reservation-2', expect.any(AbortSignal)))
    expect(await screen.findByRole('button', { name: 'Confirmar venta' })).toBeEnabled()
    resolve(persistedSale)
    await waitFor(() => expect(screen.getByRole('button', { name: 'Confirmar venta' })).toBeEnabled())
    expect(screen.queryByText('Venta registrada')).not.toBeInTheDocument()
  })

  it('maps a converted reserva to its venta', async () => {
    getReservationMock.mockResolvedValue(reservation({ estado: 'convertida', puedeConvertir: false, ventaId: 'sale-1' }))
    useLoteoMock.mockReturnValue({ status: 'loaded', loteo: { ...development, lotes: [{ ...development.lotes[0], estado: 'vendido' }] } })

    renderRoute()

    expect(await screen.findByText('Esta reserva ya se convirtió en una venta.')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Ver venta' })).toHaveAttribute('href', '/ventas/sale-1')
  })
})
