import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import PaymentConditions from './PaymentConditions'
import { EMPTY_PAYMENT_PLAN, type LotOption, type PaymentMethod, type PaymentPlanValues } from '../types'

function lot(overrides: Partial<LotOption> = {}): LotOption {
  return {
    id: 'lt-1',
    number: '7',
    blockId: 'mz-1',
    blockNumber: '1',
    developmentId: 'loteo-1',
    developmentName: 'Norte',
    state: 'disponible',
    price: 150000,
    currency: 'USD',
    area: 300,
    ...overrides,
  }
}

function renderConditions({
  method = 'contado',
  plan = EMPTY_PAYMENT_PLAN,
  lot: current = lot(),
  onMethodChange = vi.fn(),
  onPlanChange = vi.fn(),
  disabled = false,
}: {
  method?: PaymentMethod
  plan?: PaymentPlanValues
  lot?: LotOption | null
  onMethodChange?: (method: PaymentMethod) => void
  onPlanChange?: (plan: PaymentPlanValues) => void
  disabled?: boolean
} = {}) {
  return render(
    <PaymentConditions
      method={method}
      plan={plan}
      lot={current}
      onMethodChange={onMethodChange}
      onPlanChange={onPlanChange}
      disabled={disabled}
    />,
  )
}

describe('PaymentConditions', () => {
  it('starts on contado', () => {
    renderConditions()

    expect(screen.getByLabelText('Condiciones de pago')).toHaveTextContent('Contado')
  })

  it('lists the three modalidades and lets the user pick a financed one', async () => {
    const user = userEvent.setup()
    const onMethodChange = vi.fn()
    renderConditions({ onMethodChange })

    await user.click(screen.getByLabelText('Condiciones de pago'))

    expect(await screen.findByRole('option', { name: 'Contado' })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'Entrega + financiación' })).toBeInTheDocument()

    await user.click(screen.getByRole('option', { name: 'Financiado' }))
    expect(onMethodChange).toHaveBeenCalledWith('financiado')
  })

  it('shows the price of the lote as the amount, without letting it be edited', () => {
    renderConditions()

    expect(screen.getByText('Monto')).toBeInTheDocument()
    expect(screen.getByText(/150\.000/)).toBeInTheDocument()
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Monto')).not.toBeInTheDocument()
  })

  it('asks for a lote before showing an amount', () => {
    renderConditions({ lot: null })

    expect(screen.getByText('Elegí un lote para ver el monto.')).toBeInTheDocument()
  })

  it('warns when the lote has no price loaded', () => {
    renderConditions({ lot: lot({ price: null }) })

    expect(screen.getByRole('alert')).toHaveTextContent('Completá el precio del lote')
  })

  it('warns instead of showing an amount when the price is zero', () => {
    renderConditions({ lot: lot({ price: 0 }) })

    expect(screen.getByRole('alert')).toHaveTextContent('Completá el precio del lote')
    expect(screen.queryByText(/US\$\s*0/)).not.toBeInTheDocument()
  })

  it('shows the amount in the currency of the lote', () => {
    renderConditions({ lot: lot({ price: 2500, currency: 'ARS' }) })

    expect(screen.getByText(/\$\s?2\.500/)).toBeInTheDocument()
  })

  it('asks for the cuotas, the rate and the periodicidad of a financed sale', async () => {
    const user = userEvent.setup()
    const onPlanChange = vi.fn()
    renderConditions({ method: 'financiado', onPlanChange })

    expect(screen.getByText('Precio del lote')).toBeInTheDocument()
    expect(screen.getByLabelText('Periodicidad')).toHaveTextContent('Mensual')
    expect(screen.queryByLabelText('Monto de entrega')).not.toBeInTheDocument()
    expect(screen.getByText('Completá el plan para ver el detalle de las cuotas.')).toBeInTheDocument()

    await user.type(screen.getByLabelText('Cantidad de cuotas'), '1')
    expect(onPlanChange).toHaveBeenLastCalledWith({ ...EMPTY_PAYMENT_PLAN, installments: '1' })

    await user.type(screen.getByLabelText('Tasa de interés (%)'), '5')
    expect(onPlanChange).toHaveBeenLastCalledWith({ ...EMPTY_PAYMENT_PLAN, interestRate: '5' })

    await user.click(screen.getByLabelText('Periodicidad'))
    await user.click(await screen.findByRole('option', { name: 'Trimestral' }))
    expect(onPlanChange).toHaveBeenLastCalledWith({ ...EMPTY_PAYMENT_PLAN, period: 'trimestral' })
  })

  it('previews the cuotas of a financed plan with the interest applied', () => {
    renderConditions({
      method: 'financiado',
      plan: { installments: '12', interestRate: '10', period: 'mensual', downPayment: '' },
    })

    const preview = screen.getByLabelText('Detalle del plan')
    expect(within(preview).getByText('Monto financiado').nextSibling).toHaveTextContent('US$ 150.000,00')
    expect(within(preview).getByText('Cuotas').nextSibling).toHaveTextContent('12 de US$ 13.750,00')
    expect(within(preview).getByText('Total financiado').nextSibling).toHaveTextContent('US$ 165.000,00')
    expect(within(preview).queryByText('Entrega')).not.toBeInTheDocument()
  })

  it('asks for the entrega and takes it off the financed amount', async () => {
    const user = userEvent.setup()
    const onPlanChange = vi.fn()
    const plan: PaymentPlanValues = { installments: '7', interestRate: '', period: 'mensual', downPayment: '50000' }
    renderConditions({ method: 'entrega_financiada', plan, onPlanChange })

    const preview = screen.getByLabelText('Detalle del plan')
    expect(within(preview).getByText('Entrega').nextSibling).toHaveTextContent('US$ 50.000,00')
    expect(within(preview).getByText('Monto financiado').nextSibling).toHaveTextContent('US$ 100.000,00')
    // 100000 / 7 = 14285.71 x 6 + 14285.74
    expect(within(preview).getByText('Cuotas').nextSibling).toHaveTextContent(
      '7 de US$ 14.285,71 (la última de US$ 14.285,74)',
    )

    await user.type(screen.getByLabelText('Monto de entrega'), '1')
    expect(onPlanChange).toHaveBeenLastCalledWith({ ...plan, downPayment: '500001' })
  })

  it('keeps the preview quiet while the plan is invalid', () => {
    renderConditions({
      method: 'entrega_financiada',
      plan: { installments: '12', interestRate: '', period: 'mensual', downPayment: '150000' },
    })

    expect(screen.queryByLabelText('Detalle del plan')).not.toBeInTheDocument()
    expect(screen.getByText('Completá el plan para ver el detalle de las cuotas.')).toBeInTheDocument()
  })

  it('disables the plan fields with the rest of the form', () => {
    renderConditions({ method: 'entrega_financiada', disabled: true })

    expect(screen.getByLabelText('Cantidad de cuotas')).toBeDisabled()
    expect(screen.getByLabelText('Tasa de interés (%)')).toBeDisabled()
    expect(screen.getByLabelText('Monto de entrega')).toBeDisabled()
    expect(screen.getByLabelText('Periodicidad')).toBeDisabled()
  })
})
