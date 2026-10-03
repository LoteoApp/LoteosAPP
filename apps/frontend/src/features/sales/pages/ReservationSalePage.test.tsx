import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '../../../shared/api/client'
import ReservationSalePage, { type ReservationSalePageProps } from './ReservationSalePage'
import type { ReservationSaleContext, Sale } from '../types'

const context: ReservationSaleContext = {
  reservationId: 'reservation-1',
  reservationState: 'activa',
  canConvert: true,
  dueAt: '2026-10-10T15:00:00Z',
  lot: {
    id: 'lot-1', number: '7', blockId: 'block-1', blockNumber: '2', developmentId: 'loteo-1',
    developmentName: 'Las Acacias', state: 'reservado', price: 120000, currency: 'USD', area: 300,
  },
  client: { id: 'client-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' },
  seller: { id: 'seller-1', nombre: 'Beto', apellido: 'Gómez', rol: 'inmobiliaria', inmobiliariaId: 'ag-1', inmobiliariaRazonSocial: 'Inmobiliaria Sur' },
}

const sale: Sale = {
  id: 'sale-1', loteoId: 'loteo-1', loteoNombre: 'Las Acacias', loteId: 'lot-1', loteNumero: '7', manzanaNumero: '2', loteSuperficie: 300,
  cliente: context.client,
  vendedor: { id: 'seller-1', nombre: 'Beto', apellido: 'Gómez', rol: 'inmobiliaria' },
  usuarioAlta: { id: 'seller-1', nombre: 'Beto', apellido: 'Gómez', rol: 'inmobiliaria' },
  inmobiliaria: { id: 'ag-1', razonSocial: 'Inmobiliaria Sur' },
  modalidadPago: 'contado', monto: 125000, moneda: 'USD', estado: 'completada',
  fechaCreacion: '2026-10-02T15:00:00Z', fechaModificacion: '2026-10-02T15:00:00Z', reservaId: 'reservation-1',
}

function renderPage(props: Partial<ReservationSalePageProps> = {}) {
  const convert = vi.fn<ReservationSalePageProps['convert']>().mockResolvedValue(sale)
  const view = render(
    <MemoryRouter>
      <ReservationSalePage
        reservationId="reservation-1"
        context={context}
        status="loaded"
        convert={convert}
        renderPlan={<p>Plano del loteo</p>}
        {...props}
      />
    </MemoryRouter>,
  )
  return { convert, ...view }
}

afterEach(() => vi.restoreAllMocks())

describe('ReservationSalePage', () => {
  it('shows the reserva participants as fixed data and confirms with the chosen terms', async () => {
    const user = userEvent.setup()
    const { convert } = renderPage()

    const fixed = screen.getByLabelText('Datos de la reserva')
    expect(within(fixed).getByText('Las Acacias · Mz 2 · Lote 7')).toBeInTheDocument()
    expect(within(fixed).getByText('Pérez, Ana · DNI 30111222')).toBeInTheDocument()
    expect(within(fixed).getByText('Gómez, Beto · Inmobiliaria Sur')).toBeInTheDocument()
    expect(screen.queryByRole('combobox', { name: /cliente/i })).not.toBeInTheDocument()
    expect(screen.getByText('Plano del loteo')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))

    expect(convert).toHaveBeenCalledWith({ modalidadPago: 'contado' }, expect.any(String))
  })

  it('reports the persisted sale, not the previewed one, and offers its receipt and detail', async () => {
    const user = userEvent.setup()
    vi.spyOn(window, 'print').mockImplementation(() => {})
    renderPage()

    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))

    const dialog = await screen.findByRole('dialog')
    expect(dialog).toHaveTextContent('Recibo de venta')
    await user.click(within(dialog).getByRole('button', { name: 'Cerrar' }))

    expect(screen.getByText('Venta registrada')).toBeInTheDocument()
    expect(screen.getByText(/se convirtió en una venta a Pérez, Ana por US\$ 125\.000,00/)).toBeInTheDocument()
    expect(screen.getByText(/la venta quedó completada y el lote finalizado/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Ver venta' })).toHaveAttribute('href', '/ventas/sale-1')
    expect(screen.getAllByRole('link', { name: 'Volver a la reserva' })[0]).toHaveAttribute('href', '/reservas/reservation-1')
    await user.click(screen.getByRole('button', { name: 'Imprimir recibo' }))
    expect(await screen.findByRole('dialog')).toBeInTheDocument()
  })

  it('describes a financed sale as active with the lote sold', async () => {
    const user = userEvent.setup()
    renderPage({ convert: vi.fn().mockResolvedValue({ ...sale, estado: 'activa', modalidadPago: 'financiado' }) })

    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))

    expect(await screen.findByText(/La venta quedó activa y el lote vendido/)).toBeInTheDocument()
  })

  it('ignores a second click while the conversion is in flight', async () => {
    const user = userEvent.setup()
    let resolve: (value: Sale) => void = () => undefined
    const convert = vi.fn(() => new Promise<Sale>((done) => { resolve = done }))
    renderPage({ convert })

    const confirm = screen.getByRole('button', { name: 'Confirmar venta' })
    await user.click(confirm)
    await user.click(screen.getByRole('button', { name: 'Registrando venta…' }))
    expect(convert).toHaveBeenCalledTimes(1)
    resolve(sale)
    expect(await screen.findByText('Venta registrada')).toBeInTheDocument()
  })

  it('retries an uncertain conversion with the same key and locked terms', async () => {
    const user = userEvent.setup()
    const convert = vi.fn()
      .mockRejectedValueOnce(new ApiError('No hay conexión', 'network_error', 0))
      .mockResolvedValueOnce(sale)
    renderPage({ convert })

    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))

    expect(await screen.findByText(/No pudimos confirmar si la venta se registró/)).toBeInTheDocument()
    expect(screen.getByLabelText('Condiciones de pago')).toBeDisabled()
    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))

    await screen.findByText('Venta registrada')
    expect(convert.mock.calls[1][1]).toBe(convert.mock.calls[0][1])
    expect(convert.mock.calls[1][0]).toEqual(convert.mock.calls[0][0])
  })

  it('keeps the draft on a business error and lets the user refresh the reserva', async () => {
    const user = userEvent.setup()
    const onRefresh = vi.fn()
    const convert = vi.fn().mockRejectedValue(new ApiError('La reserva venció y ya no se puede convertir en venta', 'reservation_conversion_expired', 409))
    renderPage({ convert, onRefresh })

    await user.click(screen.getByLabelText('Condiciones de pago'))
    await user.click(await screen.findByRole('option', { name: 'Financiado' }))
    await user.type(screen.getByLabelText('Cantidad de cuotas'), '12')
    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))

    expect(await screen.findByText('La reserva venció y ya no se puede convertir en venta')).toBeInTheDocument()
    expect(screen.getByLabelText('Cantidad de cuotas')).toHaveValue('12')
    expect(screen.getByLabelText('Condiciones de pago')).not.toBeDisabled()
    await user.click(screen.getByRole('button', { name: 'Actualizar reserva' }))
    expect(onRefresh).toHaveBeenCalledTimes(1)
  })

  it('treats a request timeout as uncertain', async () => {
    const user = userEvent.setup()
    const convert = vi.fn().mockRejectedValue(new ApiError('Tiempo agotado', 'http_error', 408))
    renderPage({ convert })

    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))

    expect(await screen.findByText(/No pudimos confirmar si la venta se registró/)).toBeInTheDocument()
    expect(screen.getByLabelText('Condiciones de pago')).toBeDisabled()
  })

  it('retries an uncertain attempt with its own terms even after the form was rebuilt', async () => {
    const user = userEvent.setup()
    const convert = vi.fn()
      .mockRejectedValueOnce(new ApiError('No hay conexión', 'network_error', 0))
      .mockResolvedValueOnce(sale)
    const view = renderPage({ convert })

    await user.click(screen.getByLabelText('Condiciones de pago'))
    await user.click(await screen.findByRole('option', { name: 'Financiado' }))
    await user.type(screen.getByLabelText('Cantidad de cuotas'), '12')
    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))
    await screen.findByText(/No pudimos confirmar si la venta se registró/)

    // A reload in between tears the form down; it comes back on contado.
    const rerenderWith = (status: ReservationSalePageProps['status']) => view.rerender(
      <MemoryRouter>
        <ReservationSalePage reservationId="reservation-1" context={context} status={status} convert={convert} />
      </MemoryRouter>,
    )
    rerenderWith('loading')
    rerenderWith('loaded')
    expect(screen.getByLabelText('Condiciones de pago')).toHaveTextContent('Contado')
    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))

    await screen.findByText('Venta registrada')
    expect(convert.mock.calls[1]).toEqual(convert.mock.calls[0])
    expect(convert.mock.calls[1][0]).toMatchObject({ modalidadPago: 'financiado', planPago: { cantidadCuotas: 12 } })
  })

  it('refreshes the reserva after a successful conversion and reports a failed refresh without hiding the form', async () => {
    const user = userEvent.setup()
    const onRefresh = vi.fn()
    const { unmount } = renderPage({ onRefresh })
    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))
    await screen.findByText('Venta registrada')
    expect(onRefresh).toHaveBeenCalledTimes(1)
    unmount()

    renderPage({ refreshError: 'Sin conexión.' })
    expect(screen.getByText('No se pudo actualizar la reserva')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Confirmar venta' })).toBeEnabled()
  })

  it('treats an unreadable success body as uncertain', async () => {
    const user = userEvent.setup()
    renderPage({ convert: vi.fn().mockRejectedValue(new Error('No se pudo completar la operación')) })

    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))

    expect(await screen.findByText(/No pudimos confirmar si la venta se registró/)).toBeInTheDocument()
  })

  it('blocks a reserva that cannot be converted and links an existing venta', () => {
    const { unmount } = renderPage({ context: { ...context, canConvert: false } })
    expect(screen.getByText('Solo el vendedor responsable o un usuario administrativo pueden convertir una reserva vigente en venta.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Confirmar venta' })).toBeDisabled()
    unmount()

    renderPage({ context: { ...context, reservationState: 'convertida', canConvert: false, saleId: 'sale-1' } })
    expect(screen.getByText('Esta reserva ya se convirtió en una venta.')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Ver venta' })).toHaveAttribute('href', '/ventas/sale-1')
  })

  it('shows the loading, missing and failing states', () => {
    const { unmount } = renderPage({ status: 'loading', context: null })
    expect(screen.getByRole('status', { name: 'Cargando los datos para registrar la venta…' })).toBeInTheDocument()
    unmount()

    const missing = renderPage({ status: 'not-found', context: null })
    expect(screen.getByText('No encontramos esta reserva.')).toBeInTheDocument()
    missing.unmount()

    renderPage({ status: 'error', context: null, error: 'Sin conexión' })
    expect(screen.getByText('Sin conexión')).toBeInTheDocument()
  })
})
