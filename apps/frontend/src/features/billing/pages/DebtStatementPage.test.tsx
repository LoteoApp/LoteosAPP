import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '../../../shared/api/client'
import DebtStatementPage from './DebtStatementPage'
import type { DebtStatement, Payment } from '../types'

const mocks = vi.hoisted(() => ({
  getDebtStatement: vi.fn(),
  registerPayment: vi.fn(),
  settleSale: vi.fn(),
}))

vi.mock('../api/billing', () => mocks)

const statement: DebtStatement = {
  venta: {
    id: 'sale-1',
    loteoId: 'loteo-1',
    loteoNombre: 'Las Acacias',
    loteId: 'lot-1',
    loteNumero: '7',
    manzanaNumero: '2',
    loteSuperficie: 300,
    cliente: { id: 'client-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' },
    vendedor: { id: 'seller-1', nombre: 'Marta', apellido: 'Suárez', rol: 'inmobiliaria' },
    inmobiliaria: { id: 'ag-1', razonSocial: 'Inmobiliaria Sur' },
    modalidadPago: 'entrega_financiada',
    monto: 100000,
    moneda: 'USD',
    estado: 'activa',
    fechaCreacion: '2026-01-15T12:00:00Z',
    planPago: {
      id: 'plan-1',
      montoEntrega: 40000,
      cantidadCuotas: 3,
      tasaInteres: 0,
      periodicidad: 'trimestral',
      moneda: 'USD',
      montoFinanciado: 60000,
      montoCuota: 20000,
      montoTotal: 60000,
    },
  },
  entrega: { monto: 40000, estado: 'pendiente' },
  cuotas: [
    { id: 'c-1', numero: 1, monto: 20000, estado: 'vencida', fechaVencimiento: '2026-04-15T12:00:00Z' },
    { id: 'c-2', numero: 2, monto: 20000, estado: 'pendiente', fechaVencimiento: '2026-07-15T12:00:00Z' },
    { id: 'c-3', numero: 3, monto: 20000, estado: 'pendiente', fechaVencimiento: '2026-10-15T12:00:00Z' },
  ],
  resumen: {
    montoTotal: 100000,
    montoPagado: 0,
    montoPendiente: 100000,
    montoVencido: 20000,
    cuotasPagadas: 0,
    cuotasPendientes: 3,
    cuotasVencidas: 1,
    proximoVencimiento: '2026-07-15T12:00:00Z',
  },
  cobros: [],
  cargosCobrados: [],
  emitidoEl: '2026-05-01T12:00:00Z',
}

const payment: Payment = {
  id: 'cobro-1',
  ventaId: 'sale-1',
  tipo: 'pago',
  monto: 60000,
  moneda: 'USD',
  medioPago: 'transferencia',
  observacion: 'Transf. 123',
  incluyeEntrega: true,
  montoEntrega: 40000,
  cuotas: [{ id: 'c-1', numero: 1, monto: 20000, estado: 'pagada', fechaVencimiento: '2026-04-15T12:00:00Z', fechaPago: '2026-05-01T12:00:00Z', cobroId: 'cobro-1' }],
  cargos: [],
  totales: [{ moneda: 'USD', monto: 60000 }],
  usuarioAlta: { id: 'actor-1', nombre: 'Carla', apellido: 'López', rol: 'administrativo' },
  fechaPago: '2026-05-01T12:00:00Z',
  fechaCreacion: '2026-05-01T12:05:00Z',
}

// The cuota is in USD and the services in ARS: two totals, nothing converted.
const paymentWithCharges: Payment = {
  ...payment,
  id: 'cobro-2',
  incluyeEntrega: false,
  montoEntrega: 0,
  monto: 20000,
  cargos: [{ id: 'cargo-1', tipo: 'servicios', monto: 150000, moneda: 'ARS', detalle: 'Agua' }],
  totales: [
    { moneda: 'USD', monto: 20000 },
    { moneda: 'ARS', monto: 150000 },
  ],
}

const paidStatement: DebtStatement = {
  ...statement,
  entrega: { monto: 40000, estado: 'pagada', fechaPago: '2026-05-01T12:00:00Z', cobroId: 'cobro-1' },
  cuotas: [{ ...statement.cuotas[0], estado: 'pagada', fechaPago: '2026-05-01T12:00:00Z', cobroId: 'cobro-1' }, statement.cuotas[1], statement.cuotas[2]],
  resumen: { ...statement.resumen, montoPagado: 60000, montoPendiente: 40000, montoVencido: 0, cuotasPagadas: 1, cuotasPendientes: 2, cuotasVencidas: 0 },
  cobros: [payment],
  cargosCobrados: [],
}

const paidWithChargesStatement: DebtStatement = {
  ...paidStatement,
  cobros: [paymentWithCharges],
  cargosCobrados: [{ moneda: 'ARS', monto: 150000 }],
}

function renderPage(token = 'token') {
  return render(
    <MemoryRouter initialEntries={['/cobranzas/sale-1']}>
      <Routes>
        <Route path="/cobranzas/:ventaId" element={<DebtStatementPage accessToken={token} />} />
      </Routes>
    </MemoryRouter>,
  )
}

afterEach(() => vi.resetAllMocks())

describe('DebtStatementPage', () => {
  it('shows the sale, the summary, the cuotas and the printable statement', async () => {
    mocks.getDebtStatement.mockResolvedValue(statement)
    const user = userEvent.setup()
    const print = vi.spyOn(window, 'print').mockImplementation(() => {})

    renderPage()

    expect(screen.getByRole('status')).toHaveTextContent('Cargando estado de deuda…')
    expect(await screen.findByText('Pérez, Ana')).toBeInTheDocument()
    expect(mocks.getDebtStatement).toHaveBeenCalledWith('token', 'sale-1', expect.anything())
    expect(screen.getByText('Las Acacias · Mz 2 · Lote 7 · DNI 30111222')).toBeInTheDocument()
    expect(screen.getByText('Entrega + financiación')).toBeInTheDocument()
    expect(screen.getByText('Suárez, Marta (Inmobiliaria Sur)')).toBeInTheDocument()
    expect(screen.getByText('Total del plan')).toBeInTheDocument()
    expect(screen.getByText('Saldo pendiente')).toBeInTheDocument()
    expect(screen.getByText('3 cuotas pendientes')).toBeInTheDocument()
    expect(screen.getByText('1 cuota vencida')).toBeInTheDocument()

    const table = screen.getByRole('table', { name: 'Cuotas' })
    expect(within(table).getByText('Entrega')).toBeInTheDocument()
    expect(within(table).getByText('Cuota 3')).toBeInTheDocument()
    expect(within(table).getByText('Vencida')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Volver a cobranzas' })).toHaveAttribute('href', '/cobranzas')
    expect(screen.getByRole('link', { name: 'Ver venta' })).toHaveAttribute('href', '/ventas/sale-1')
    expect(screen.getByText('Todavía no se registraron cobros sobre esta venta.')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Imprimir estado de deuda' }))
    const dialog = await screen.findByRole('dialog')
    expect(dialog).toHaveTextContent('Estado de deuda')
    expect(dialog).toHaveTextContent('3 cuotas trimestral de')
    expect(within(dialog).getByRole('table', { name: 'Detalle de cuotas' })).toBeInTheDocument()
    await user.click(within(dialog).getByRole('button', { name: 'Imprimir estado de deuda' }))
    expect(print).toHaveBeenCalledTimes(1)
    await user.click(within(dialog).getByRole('button', { name: 'Cerrar' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  })

  it('collects the selected cuotas in order and shows the receipt', async () => {
    mocks.getDebtStatement.mockResolvedValueOnce(statement).mockResolvedValueOnce(paidStatement)
    mocks.registerPayment.mockResolvedValue(payment)
    const user = userEvent.setup()
    const print = vi.spyOn(window, 'print').mockImplementation(() => {})

    renderPage()
    await screen.findByText('Pérez, Ana')

    const collect = screen.getByRole('button', { name: /Registrar cobro/ })
    expect(collect).toBeDisabled()

    // Clicking cuota 2 takes cuota 1 with it; clicking it again drops it.
    await user.click(screen.getByRole('checkbox', { name: 'Cobrar la cuota 2' }))
    expect(screen.getByRole('checkbox', { name: 'Cobrar la cuota 1' })).toBeChecked()
    expect(screen.getByRole('checkbox', { name: 'Cobrar la cuota 2' })).toBeChecked()
    await user.click(screen.getByRole('checkbox', { name: 'Cobrar la cuota 2' }))
    expect(screen.getByRole('checkbox', { name: 'Cobrar la cuota 2' })).not.toBeChecked()
    await user.click(screen.getByRole('checkbox', { name: 'Cobrar la entrega' }))
    expect(collect).toBeEnabled()
    expect(collect).toHaveTextContent(/US\$\s?60\.000,00/)

    await user.click(collect)
    const dialog = await screen.findByRole('dialog', { name: 'Registrar cobro' })
    expect(dialog).toHaveTextContent('Entrega + Cuota 1')
    await user.click(within(dialog).getByRole('combobox', { name: 'Medio de pago' }))
    await user.click(await screen.findByRole('option', { name: 'Transferencia' }))
    expect(within(dialog).getByRole('combobox', { name: 'Medio de pago' })).toHaveTextContent('Transferencia')
    await user.clear(within(dialog).getByLabelText('Fecha de pago'))
    await user.type(within(dialog).getByLabelText('Fecha de pago'), '2026-05-01')
    await user.click(within(dialog).getByLabelText('Observación'))
    await user.paste('Transf. 123')
    await user.click(within(dialog).getByRole('button', { name: 'Confirmar cobro' }))

    await waitFor(() =>
      expect(mocks.registerPayment).toHaveBeenCalledWith('token', 'sale-1', {
        cuotaIds: ['c-1'],
        incluirEntrega: true,
        cargos: [],
        medioPago: 'transferencia',
        fechaPago: '2026-05-01',
        observacion: 'Transf. 123',
      }),
    )
    const receipt = await screen.findByRole('dialog', { name: 'Recibo de cobro' })
    expect(receipt).toHaveTextContent('Entrega + Cuota 1')
    expect(receipt).toHaveTextContent('Transferencia')
    expect(receipt).toHaveTextContent('Transf. 123')
    expect(receipt).toHaveTextContent('López, Carla')
    await user.click(within(receipt).getByRole('button', { name: 'Imprimir recibo' }))
    expect(print).toHaveBeenCalledTimes(1)
    await user.click(within(receipt).getByRole('button', { name: 'Cerrar' }))

    await waitFor(() => expect(mocks.getDebtStatement).toHaveBeenCalledTimes(2))
    expect(await screen.findByText('1 cobro, del más reciente al más antiguo.')).toBeInTheDocument()
    expect(screen.queryByRole('checkbox', { name: 'Cobrar la entrega' })).not.toBeInTheDocument()
    expect(screen.queryByRole('checkbox', { name: 'Cobrar la cuota 1' })).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /Imprimir recibo del cobro del 01\/05\/2026/ }))
    expect(await screen.findByRole('dialog', { name: 'Recibo de cobro' })).toHaveTextContent('Cuota 1')
  }, 15000)

  it('collects extra charges in another currency without mixing them with the cuota', async () => {
    mocks.getDebtStatement.mockResolvedValueOnce(statement).mockResolvedValueOnce(paidWithChargesStatement)
    mocks.registerPayment.mockResolvedValue(paymentWithCharges)
    const user = userEvent.setup()

    renderPage()
    await screen.findByText('Pérez, Ana')
    await user.click(screen.getByRole('checkbox', { name: 'Cobrar la cuota 1' }))
    await user.click(screen.getByRole('button', { name: /Registrar cobro/ }))
    const dialog = await screen.findByRole('dialog', { name: 'Registrar cobro' })

    await user.click(within(dialog).getByRole('button', { name: 'Agregar cargo' }))
    // A cargo is typed fresh every time: the browser must not offer what was
    // saved for another one.
    expect(within(dialog).getByLabelText('Detalle')).toHaveAttribute('autocomplete', 'off')
    expect(within(dialog).getByRole('combobox', { name: 'Tipo' })).toHaveTextContent('Servicios')
    await user.click(within(dialog).getByLabelText('Monto'))
    await user.paste('150000')
    // The currency starts on the sale's and the list offers the usual ones.
    const currency = within(dialog).getByRole('combobox', { name: 'Moneda' })
    expect(currency).toHaveTextContent('USD')
    await user.click(currency)
    await user.click(await screen.findByRole('option', { name: 'ARS' }))
    await user.click(within(dialog).getByLabelText('Detalle'))
    await user.paste('Agua')

    // es-AR renders ARS as a bare "$"; the currency each amount belongs to is
    // the sr-only term next to it.
    const totals = within(dialog).getByRole('group', { name: 'Total a cobrar' })
    expect(totals).toHaveTextContent(/US\$\s?20\.000,00/)
    expect(totals).toHaveTextContent(/ARS\$\s?150\.000,00/)

    await user.click(within(dialog).getByRole('button', { name: 'Confirmar cobro' }))

    await waitFor(() =>
      expect(mocks.registerPayment).toHaveBeenCalledWith(
        'token',
        'sale-1',
        expect.objectContaining({
          cuotaIds: ['c-1'],
          cargos: [{ tipo: 'servicios', monto: 150000, moneda: 'ARS', detalle: 'Agua' }],
        }),
      ),
    )
    const receipt = await screen.findByRole('dialog', { name: 'Recibo de cobro' })
    expect(receipt).toHaveTextContent('Servicios · Agua')
    const collected = within(receipt).getByRole('group', { name: 'Importe cobrado' })
    expect(collected).toHaveTextContent(/US\$\s?20\.000,00/)
    expect(collected).toHaveTextContent(/ARS\$\s?150\.000,00/)
    await user.click(within(receipt).getByRole('button', { name: 'Cerrar' }))

    expect(await screen.findByText(/Cargos adicionales cobrados: \$\s?150\.000,00/)).toBeInTheDocument()
    expect(screen.getByText(/Servicios · Agua: \$\s?150\.000,00/)).toBeInTheDocument()
  }, 15000)

  it('rejects a charge without an amount before sending the cobro', async () => {
    mocks.getDebtStatement.mockResolvedValue(statement)
    const user = userEvent.setup()

    renderPage()
    await screen.findByText('Pérez, Ana')
    await user.click(screen.getByRole('checkbox', { name: 'Cobrar la cuota 1' }))
    await user.click(screen.getByRole('button', { name: /Registrar cobro/ }))
    const dialog = await screen.findByRole('dialog', { name: 'Registrar cobro' })

    await user.click(within(dialog).getByRole('button', { name: 'Agregar cargo' }))
    await user.click(within(dialog).getByRole('button', { name: 'Agregar cargo' }))
    // Typing in the second row leaves the first one as it was.
    await user.click(within(dialog).getAllByLabelText('Monto')[1])
    await user.paste('500')
    expect(within(dialog).getAllByLabelText('Monto')[0]).toHaveValue('')

    await user.click(within(dialog).getByRole('button', { name: 'Confirmar cobro' }))
    expect(
      await within(dialog).findByText('Ingresá el monto de "Servicios", mayor a cero y con hasta 2 decimales.'),
    ).toBeInTheDocument()
    expect(mocks.registerPayment).not.toHaveBeenCalled()

    await user.click(within(dialog).getByRole('button', { name: 'Quitar el cargo 1' }))
    expect(within(dialog).getAllByLabelText('Monto')).toHaveLength(1)
    expect(within(dialog).getByLabelText('Monto')).toHaveValue('500')
    await user.click(within(dialog).getByRole('button', { name: 'Quitar el cargo 1' }))
    expect(within(dialog).queryByLabelText('Monto')).not.toBeInTheDocument()
  }, 15000)

  it('validates the terms and shows the backend error without losing the dialog', async () => {
    mocks.getDebtStatement.mockResolvedValue(statement)
    mocks.registerPayment.mockRejectedValue(new ApiError('Alguna de las cuotas seleccionadas ya está pagada', 'payment_installment_paid', 409))
    const user = userEvent.setup()

    renderPage()
    await screen.findByText('Pérez, Ana')
    await user.click(screen.getByRole('checkbox', { name: 'Cobrar la cuota 1' }))
    await user.click(screen.getByRole('button', { name: /Registrar cobro/ }))
    const dialog = await screen.findByRole('dialog', { name: 'Registrar cobro' })

    await user.clear(within(dialog).getByLabelText('Fecha de pago'))
    await user.type(within(dialog).getByLabelText('Fecha de pago'), '2999-01-01')
    await user.click(within(dialog).getByRole('button', { name: 'Confirmar cobro' }))
    expect(await within(dialog).findByText('La fecha de pago no puede ser futura.')).toBeInTheDocument()
    expect(mocks.registerPayment).not.toHaveBeenCalled()

    await user.clear(within(dialog).getByLabelText('Fecha de pago'))
    await user.click(within(dialog).getByRole('button', { name: 'Confirmar cobro' }))
    expect(await within(dialog).findByText('Alguna de las cuotas seleccionadas ya está pagada')).toBeInTheDocument()
    expect(mocks.registerPayment).toHaveBeenCalledWith('token', 'sale-1', expect.objectContaining({ cuotaIds: ['c-1'], fechaPago: '' }))
    expect(screen.getByRole('dialog', { name: 'Registrar cobro' })).toBeInTheDocument()

    await user.click(within(dialog).getByRole('button', { name: 'Cancelar' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  }, 15000)

  it('settles the whole balance in one cobro', async () => {
    const settled: DebtStatement = {
      ...paidStatement,
      venta: { ...statement.venta, estado: 'completada' },
      cuotas: paidStatement.cuotas.map((cuota) => ({ ...cuota, estado: 'pagada' as const, fechaPago: '2026-05-02T12:00:00Z' })),
      resumen: { ...paidStatement.resumen, montoPagado: 100000, montoPendiente: 0, cuotasPagadas: 3, cuotasPendientes: 0, proximoVencimiento: undefined },
      cobros: [
        {
          ...payment,
          id: 'cobro-2',
          tipo: 'cancelacion_total',
          monto: 40000,
          incluyeEntrega: false,
          montoEntrega: 0,
          observacion: undefined,
          cuotas: statement.cuotas.slice(1),
          cargos: [],
          totales: [{ moneda: 'USD', monto: 40000 }],
        },
        payment,
      ],
    }
    mocks.getDebtStatement.mockResolvedValueOnce(paidStatement).mockResolvedValueOnce(settled)
    mocks.settleSale.mockResolvedValue(settled.cobros[0])
    const user = userEvent.setup()

    renderPage()
    await screen.findByText('Pérez, Ana')
    await user.click(screen.getByRole('button', { name: 'Cancelar saldo total' }))
    const dialog = await screen.findByRole('dialog', { name: 'Cancelar el saldo total' })
    expect(dialog).toHaveTextContent(/US\$\s?40\.000,00/)
    expect(dialog).toHaveTextContent('Todo el saldo pendiente')
    await user.click(within(dialog).getByRole('button', { name: 'Confirmar cancelación total' }))

    await waitFor(() =>
      expect(mocks.settleSale).toHaveBeenCalledWith('token', 'sale-1', expect.objectContaining({ montoEsperado: 40000, medioPago: 'efectivo' })),
    )
    const receipt = await screen.findByRole('dialog', { name: 'Recibo de cobro' })
    expect(receipt).toHaveTextContent('Cancelación total')
    expect(receipt).toHaveTextContent('Cuotas 2 a 3')
    await user.click(within(receipt).getByRole('button', { name: 'Cerrar' }))

    expect(await screen.findByText('Completada')).toBeInTheDocument()
    expect(screen.getByText(/La venta está completada; no admite nuevos cobros./)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Registrar cobro/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('checkbox')).not.toBeInTheDocument()
    expect(screen.getByText('Sin cuotas pendientes')).toBeInTheDocument()
    expect(screen.getByText('2 cobros, del más reciente al más antiguo.')).toBeInTheDocument()
  })

  it('shows the load error and waits for a token', async () => {
    mocks.getDebtStatement.mockRejectedValueOnce(new ApiError('La venta solicitada no existe', 'sale_not_found', 404))

    const { unmount } = renderPage()
    expect(await screen.findByText('La venta solicitada no existe')).toBeInTheDocument()
    expect(screen.getByText('No se pudo cargar el estado de deuda')).toBeInTheDocument()
    unmount()

    mocks.getDebtStatement.mockClear()
    renderPage('')
    expect(mocks.getDebtStatement).not.toHaveBeenCalled()
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })
})
