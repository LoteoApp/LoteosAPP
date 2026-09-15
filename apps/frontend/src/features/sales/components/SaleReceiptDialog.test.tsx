import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import SaleReceiptDialog from './SaleReceiptDialog'
import type { SaleReceipt } from '../types'

function receipt(overrides: Partial<SaleReceipt> = {}): SaleReceipt {
  return {
    emitidoEl: '2026-09-07T10:00:00.000Z',
    lote: {
      id: 'lt-1',
      numero: '7',
      manzanaId: 'mz-1',
      manzanaNumero: '1',
      loteoId: 'loteo-1',
      loteoNombre: 'Norte',
      estado: 'disponible',
      precio: 150000,
      moneda: 'USD',
      superficie: 300,
    },
    cliente: { id: 'cl-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' },
    seller: {
      id: 'us-1',
      nombre: 'Marta',
      apellido: 'Suárez',
      rol: 'inmobiliaria',
      inmobiliariaId: 'ag-1',
      inmobiliariaRazonSocial: 'Inmobiliaria Sur',
    },
    method: 'contado',
    monto: 150000,
    moneda: 'USD',
    ...overrides,
  }
}

describe('SaleReceiptDialog', () => {
  it('shows the data of the sale in the receipt', () => {
    render(<SaleReceiptDialog open receipt={receipt()} onClose={vi.fn()} />)

    const dialog = screen.getByRole('dialog')
    expect(dialog).toHaveTextContent('Recibo de venta')
    expect(dialog).toHaveTextContent('Norte · Mz 1 · Lote 7')
    expect(dialog).toHaveTextContent('300 m²')
    expect(dialog).toHaveTextContent('Pérez, Ana')
    expect(dialog).toHaveTextContent('30111222')
    expect(dialog).toHaveTextContent('Suárez, Marta')
    expect(dialog).toHaveTextContent('Inmobiliaria Sur')
    expect(dialog).toHaveTextContent('Contado')
    expect(dialog).toHaveTextContent('US$ 150.000,00')
    expect(dialog).toHaveTextContent('07/09/2026')
  })

  it('prints the plan of a financed sale', () => {
    render(
      <SaleReceiptDialog
        open
        receipt={receipt({
          method: 'entrega_financiada',
          plan: {
            montoEntrega: 50000,
            cantidadCuotas: 10,
            tasaInteres: 12.5,
            periodicidad: 'mensual',
            montoFinanciado: 100000,
            montoCuota: 11250,
            montoTotal: 112500,
          },
        })}
        onClose={vi.fn()}
      />,
    )

    const dialog = screen.getByRole('dialog')
    expect(dialog).toHaveTextContent('Entrega + financiación')
    const plan = screen.getByLabelText('Plan de pago')
    expect(plan).toHaveTextContent('Entrega')
    expect(plan).toHaveTextContent('US$ 50.000,00')
    expect(plan).toHaveTextContent('Monto financiado')
    expect(plan).toHaveTextContent('US$ 100.000,00')
    expect(plan).toHaveTextContent('10 × US$ 11.250,00 · mensual')
    expect(plan).toHaveTextContent('12,5 %')
    expect(plan).toHaveTextContent('US$ 112.500,00')
  })

  it('omits the entrega row of a plan without one and the plan of a contado sale', () => {
    const { rerender } = render(
      <SaleReceiptDialog
        open
        receipt={receipt({
          method: 'financiado',
          plan: {
            montoEntrega: 0,
            cantidadCuotas: 3,
            tasaInteres: 0,
            periodicidad: 'trimestral',
            montoFinanciado: 150000,
            montoCuota: 50000,
            montoTotal: 150000,
          },
        })}
        onClose={vi.fn()}
      />,
    )

    expect(screen.getByLabelText('Plan de pago')).not.toHaveTextContent('Entrega')
    expect(screen.getByLabelText('Plan de pago')).toHaveTextContent('3 × US$ 50.000,00 · trimestral')

    rerender(<SaleReceiptDialog open receipt={receipt()} onClose={vi.fn()} />)
    expect(screen.queryByLabelText('Plan de pago')).not.toBeInTheDocument()
  })

  it('reads a sale without inmobiliaria as a direct sale', () => {
    render(
      <SaleReceiptDialog
        open
        receipt={receipt({
          seller: { id: 'us-2', nombre: 'Sofía', apellido: 'Luna', rol: 'administrativo' },
        })}
        onClose={vi.fn()}
      />,
    )

    expect(screen.getByRole('dialog')).toHaveTextContent('Venta directa')
  })

  it('omits the superficie when the lote has none', () => {
    const withoutArea = receipt()
    render(
      <SaleReceiptDialog
        open
        receipt={{ ...withoutArea, lote: { ...withoutArea.lote, superficie: null } }}
        onClose={vi.fn()}
      />,
    )

    expect(screen.getByRole('dialog')).not.toHaveTextContent('Superficie')
  })

  it('prints the receipt', async () => {
    const user = userEvent.setup()
    const print = vi.spyOn(window, 'print').mockImplementation(() => {})
    render(<SaleReceiptDialog open receipt={receipt()} onClose={vi.fn()} />)

    await user.click(screen.getByRole('button', { name: 'Imprimir recibo' }))

    expect(print).toHaveBeenCalledTimes(1)
    print.mockRestore()
  })

  it('closes the receipt', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    render(<SaleReceiptDialog open receipt={receipt()} onClose={onClose} />)

    await user.click(screen.getByRole('button', { name: 'Cerrar' }))

    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('closes the receipt with the escape key', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    render(<SaleReceiptDialog open receipt={receipt()} onClose={onClose} />)

    await user.keyboard('{Escape}')

    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('renders nothing without a receipt', () => {
    render(<SaleReceiptDialog open={false} receipt={null} onClose={vi.fn()} />)

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })
})
