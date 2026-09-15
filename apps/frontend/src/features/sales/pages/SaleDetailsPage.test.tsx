import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '../../../shared/api/client'
import SaleDetailsPage from './SaleDetailsPage'
import type { Sale } from '../types'

const getSaleMock = vi.hoisted(() => vi.fn())

vi.mock('../api/sales', () => ({ getSale: getSaleMock }))

const sale: Sale = {
  id: 'sale-1',
  loteoId: 'loteo-1',
  loteoNombre: 'Las Acacias',
  loteId: 'lot-1',
  loteNumero: '7',
  manzanaNumero: '2',
  loteSuperficie: 300,
  cliente: { id: 'client-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' },
  vendedor: { id: 'seller-1', nombre: 'Marta', apellido: 'Suárez', rol: 'inmobiliaria' },
  usuarioAlta: { id: 'actor-1', nombre: 'Carla', apellido: 'López', rol: 'administrativo' },
  inmobiliaria: { id: 'ag-1', razonSocial: 'Inmobiliaria Sur' },
  modalidadPago: 'contado',
  monto: 120000,
  moneda: 'USD',
  estado: 'activa',
  fechaCreacion: '2026-09-14T15:00:00Z',
  fechaModificacion: '2026-09-14T15:00:00Z',
}

function renderPage(token = 'token') {
  return render(
    <MemoryRouter initialEntries={['/ventas/sale-1']}>
      <Routes>
        <Route
          path="/ventas/:id"
          element={<SaleDetailsPage accessToken={token} renderPlan={(loaded) => <p>Plano de {loaded.loteoNombre}</p>} />}
        />
      </Routes>
    </MemoryRouter>,
  )
}

afterEach(() => vi.clearAllMocks())

describe('SaleDetailsPage', () => {
  it('loads the sale and prints its receipt', async () => {
    getSaleMock.mockResolvedValue(sale)
    const user = userEvent.setup()
    const print = vi.spyOn(window, 'print').mockImplementation(() => {})

    renderPage()

    expect(screen.getByRole('status', { name: 'Cargando ventas…' })).toBeInTheDocument()
    expect(await screen.findByText('Las Acacias')).toBeInTheDocument()
    expect(getSaleMock).toHaveBeenCalledWith('token', 'sale-1', expect.anything())
    expect(screen.getByText('Plano de Las Acacias')).toBeInTheDocument()
    expect(screen.getByText('Manzana 2 · Lote 7 · 300 m²')).toBeInTheDocument()
    expect(screen.getByText('Pérez, Ana · DNI 30111222')).toBeInTheDocument()
    expect(screen.getByText('Suárez, Marta')).toBeInTheDocument()
    expect(screen.getByText('Inmobiliaria Sur')).toBeInTheDocument()
    expect(screen.getByText('López, Carla')).toBeInTheDocument()
    expect(screen.getByText('Contado')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Volver a ventas' })).toHaveAttribute('href', '/ventas')
    expect(screen.getByRole('link', { name: 'Ver loteo' })).toHaveAttribute('href', '/lotes/loteo-1')

    await user.click(screen.getByRole('button', { name: 'Imprimir recibo' }))
    const dialog = await screen.findByRole('dialog')
    expect(dialog).toHaveTextContent('Recibo de venta')
    expect(dialog).toHaveTextContent('Las Acacias · Mz 2 · Lote 7')
    await user.click(within(dialog).getByRole('button', { name: 'Imprimir recibo' }))
    expect(print).toHaveBeenCalledTimes(1)
    print.mockRestore()
  })

  it('shows the plan and the cuotas of a financed sale', async () => {
    const user = userEvent.setup()
    getSaleMock.mockResolvedValue({
      ...sale,
      modalidadPago: 'entrega_financiada',
      planPago: {
        id: 'plan-1',
        montoEntrega: 20000,
        cantidadCuotas: 2,
        tasaInteres: 10,
        periodicidad: 'bimestral',
        moneda: 'USD',
        montoFinanciado: 100000,
        montoCuota: 55000,
        montoTotal: 110000,
        cuotas: [
          { id: 'c-1', numero: 1, monto: 55000, estado: 'pagada', fechaVencimiento: '2026-11-14T15:00:00Z', fechaPago: '2026-11-10T12:00:00Z' },
          { id: 'c-2', numero: 2, monto: 55000, estado: 'pendiente', fechaVencimiento: '2027-01-14T15:00:00Z' },
        ],
      },
    })

    renderPage()

    expect(await screen.findByText('Entrega + financiación')).toBeInTheDocument()
    expect(screen.getByText('Plan de pago')).toBeInTheDocument()
    expect(screen.getByText('2 cuotas · bimestral')).toBeInTheDocument()
    expect(screen.getByText('Entrega').nextSibling).toHaveTextContent('US$ 20.000,00')
    expect(screen.getByText('Monto financiado').nextSibling).toHaveTextContent('US$ 100.000,00')
    expect(screen.getByText('Tasa de interés').nextSibling).toHaveTextContent('10 %')
    expect(screen.getByText('Monto por cuota').nextSibling).toHaveTextContent('US$ 55.000,00')
    expect(screen.getByText('Total financiado').nextSibling).toHaveTextContent('US$ 110.000,00')

    const table = screen.getByRole('table', { name: 'Cuotas' })
    const rows = within(table).getAllByRole('row')
    expect(rows).toHaveLength(3)
    expect(rows[1]).toHaveTextContent('1')
    expect(rows[1]).toHaveTextContent('14/11/2026')
    expect(rows[1]).toHaveTextContent('US$ 55.000,00')
    expect(rows[1]).toHaveTextContent('Pagada')
    expect(rows[2]).toHaveTextContent('14/01/2027')
    expect(rows[2]).toHaveTextContent('Pendiente')

    await user.click(screen.getByRole('button', { name: 'Imprimir recibo' }))
    expect(await screen.findByRole('dialog')).toHaveTextContent('2 × US$ 55.000,00 · bimestral')
  })

  it('describes a financed plan without entrega or cuotas', async () => {
    getSaleMock.mockResolvedValue({
      ...sale,
      modalidadPago: 'financiado',
      planPago: {
        id: 'plan-1',
        montoEntrega: 0,
        cantidadCuotas: 1,
        tasaInteres: 0,
        periodicidad: 'mensual',
        moneda: 'USD',
        montoFinanciado: 120000,
        montoCuota: 120000,
        montoTotal: 120000,
      },
    })

    renderPage()

    expect(await screen.findByText('1 cuota · mensual')).toBeInTheDocument()
    expect(screen.queryByText('Entrega')).not.toBeInTheDocument()
    expect(screen.queryByRole('table', { name: 'Cuotas' })).not.toBeInTheDocument()
  })

  it('describes a venta directa of a lote without numbers or surface', async () => {
    getSaleMock.mockResolvedValue({ ...sale, inmobiliaria: undefined, loteNumero: '', manzanaNumero: '', loteSuperficie: null, estado: 'cancelada' })

    renderPage()

    expect(await screen.findByText('Lote sin número')).toBeInTheDocument()
    expect(screen.getByText('Venta directa')).toBeInTheDocument()
    expect(screen.getByText('Cancelada')).toBeInTheDocument()
  })

  it('reloads when the session changes', async () => {
    getSaleMock.mockResolvedValue(sale)
    const { rerender } = render(
      <MemoryRouter initialEntries={['/ventas/sale-1']}>
        <Routes>
          <Route path="/ventas/:id" element={<SaleDetailsPage accessToken="token" />} />
        </Routes>
      </MemoryRouter>,
    )
    expect(await screen.findByText('Las Acacias')).toBeInTheDocument()

    rerender(
      <MemoryRouter initialEntries={['/ventas/sale-1']}>
        <Routes>
          <Route path="/ventas/:id" element={<SaleDetailsPage accessToken="token-2" />} />
        </Routes>
      </MemoryRouter>,
    )
    await waitFor(() => expect(getSaleMock).toHaveBeenLastCalledWith('token-2', 'sale-1', expect.anything()))
    expect(await screen.findByText('Las Acacias')).toBeInTheDocument()
  })

  it('shows the backend error when the sale cannot be loaded', async () => {
    getSaleMock.mockRejectedValue(new ApiError('La venta solicitada no existe', 'sale_not_found', 404))

    renderPage()

    expect(await screen.findByText('No se pudo cargar la venta')).toBeInTheDocument()
    expect(screen.getByText('La venta solicitada no existe')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Imprimir recibo' })).not.toBeInTheDocument()
  })

  it('does not request the sale without a session', async () => {
    renderPage('')

    await waitFor(() => expect(screen.getByRole('link', { name: 'Volver a ventas' })).toBeInTheDocument())
    expect(getSaleMock).not.toHaveBeenCalled()
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })
})
