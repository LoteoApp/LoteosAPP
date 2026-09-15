import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import PaymentConditions from './PaymentConditions'
import type { LotOption } from '../types'

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

describe('PaymentConditions', () => {
  it('starts on contado', () => {
    render(<PaymentConditions method="contado" lot={lot()} onMethodChange={vi.fn()} />)

    expect(screen.getByLabelText('Condiciones de pago')).toHaveTextContent('Contado')
  })

  it('lists the three modalidades, with the pending ones unavailable', async () => {
    const user = userEvent.setup()
    const onMethodChange = vi.fn()
    render(<PaymentConditions method="contado" lot={lot()} onMethodChange={onMethodChange} />)

    await user.click(screen.getByLabelText('Condiciones de pago'))

    expect(await screen.findByRole('option', { name: 'Contado' })).not.toHaveAttribute(
      'aria-disabled',
      'true',
    )

    const financiado = screen.getByRole('option', { name: 'Financiado (próximamente)' })
    const entrega = screen.getByRole('option', { name: 'Entrega + financiación (próximamente)' })
    expect(financiado).toHaveAttribute('aria-disabled', 'true')
    expect(entrega).toHaveAttribute('aria-disabled', 'true')

    await user.click(financiado)
    expect(onMethodChange).not.toHaveBeenCalled()
  })

  it('shows the price of the lote as the amount, without letting it be edited', () => {
    render(<PaymentConditions method="contado" lot={lot()} onMethodChange={vi.fn()} />)

    expect(screen.getByText('Monto')).toBeInTheDocument()
    expect(screen.getByText(/150\.000/)).toBeInTheDocument()
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Monto')).not.toBeInTheDocument()
  })

  it('asks for a lote before showing an amount', () => {
    render(<PaymentConditions method="contado" lot={null} onMethodChange={vi.fn()} />)

    expect(screen.getByText('Elegí un lote para ver el monto.')).toBeInTheDocument()
  })

  it('warns when the lote has no price loaded', () => {
    render(
      <PaymentConditions method="contado" lot={lot({ price: null })} onMethodChange={vi.fn()} />,
    )

    expect(screen.getByRole('alert')).toHaveTextContent('Completá el precio del lote')
  })

  it('warns instead of showing an amount when the price is zero', () => {
    render(
      <PaymentConditions method="contado" lot={lot({ price: 0 })} onMethodChange={vi.fn()} />,
    )

    expect(screen.getByRole('alert')).toHaveTextContent('Completá el precio del lote')
    expect(screen.queryByText(/US\$\s*0/)).not.toBeInTheDocument()
  })

  it('shows the amount in the currency of the lote', () => {
    render(
      <PaymentConditions
        method="contado"
        lot={lot({ price: 2500, currency: 'ARS' })}
        onMethodChange={vi.fn()}
      />,
    )

    expect(screen.getByText(/\$\s?2\.500/)).toBeInTheDocument()
  })

  it('hides the amount on a modalidad that is not contado', () => {
    render(<PaymentConditions method="financiado" lot={lot()} onMethodChange={vi.fn()} />)

    expect(screen.queryByText('Monto')).not.toBeInTheDocument()
  })
})
