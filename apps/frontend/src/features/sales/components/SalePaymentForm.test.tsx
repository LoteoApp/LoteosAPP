import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import SalePaymentForm from './SalePaymentForm'
import type { ClientOption, LotOption, SalePaymentTerms, SellerOption } from '../types'

const lot: LotOption = {
  id: 'lot-1', number: '7', blockId: 'block-1', blockNumber: '2', developmentId: 'loteo-1',
  developmentName: 'Las Acacias', state: 'reservado', price: 120000, currency: 'USD', area: 300,
}
const client: ClientOption = { id: 'client-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' }
const seller: SellerOption = { id: 'seller-1', nombre: 'Beto', apellido: 'Gómez', rol: 'administrativo' }

function renderForm(props: Partial<Parameters<typeof SalePaymentForm>[0]> = {}) {
  const onSubmit = vi.fn<(terms: SalePaymentTerms) => Promise<boolean>>().mockResolvedValue(true)
  render(
    <SalePaymentForm
      lot={lot}
      client={client}
      seller={seller}
      participants={<p>Datos fijos de la reserva</p>}
      onSubmit={onSubmit}
      {...props}
    />,
  )
  return onSubmit
}

describe('SalePaymentForm', () => {
  it('shows the participants and confirms contado with the lote price', async () => {
    const user = userEvent.setup()
    const onSubmit = renderForm()

    expect(screen.getByText('Datos fijos de la reserva')).toBeInTheDocument()
    expect(screen.getByText('US$ 120.000,00')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))

    expect(onSubmit).toHaveBeenCalledWith({ modalidadPago: 'contado' })
  })

  it('sends the parsed plan of a financed sale', async () => {
    const user = userEvent.setup()
    const onSubmit = renderForm()

    await user.click(screen.getByLabelText('Condiciones de pago'))
    await user.click(await screen.findByRole('option', { name: 'Entrega + financiación' }))
    await user.type(screen.getByLabelText('Monto de entrega'), '20000')
    await user.type(screen.getByLabelText('Cantidad de cuotas'), '10')
    expect(screen.getByLabelText('Detalle del plan')).toHaveTextContent('Monto financiado')
    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))

    expect(onSubmit).toHaveBeenCalledWith({
      modalidadPago: 'entrega_financiada',
      planPago: { cantidadCuotas: 10, tasaInteres: 0, periodicidad: 'mensual', montoEntrega: 20000 },
    })
  })

  it('explains what is missing and keeps the confirmation off', () => {
    renderForm({ client: null })

    expect(screen.getByText('Elegí el cliente comprador.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Confirmar venta' })).toBeDisabled()
  })

  it('locks the terms but still allows the retry of an uncertain request', async () => {
    const user = userEvent.setup()
    const onSubmit = renderForm({ termsLocked: true, error: 'No pudimos confirmar si la venta se registró.' })

    expect(screen.getByText('No pudimos confirmar si la venta se registró.')).toBeInTheDocument()
    expect(screen.getByLabelText('Condiciones de pago')).toBeDisabled()
    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))
    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1))
  })

  it('disables everything while the sale is being registered', () => {
    renderForm({ isSubmitting: true })

    expect(screen.getByRole('button', { name: 'Registrando venta…' })).toBeDisabled()
    expect(screen.getByLabelText('Condiciones de pago')).toBeDisabled()
  })
})
