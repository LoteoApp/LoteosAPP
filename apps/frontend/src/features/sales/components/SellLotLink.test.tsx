import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { describe, expect, it } from 'vitest'
import SellLotLink from './SellLotLink'

function renderLink(lot: { id: string; number: string; price: number | null; currency: string }) {
  render(
    <MemoryRouter>
      <SellLotLink developmentId="loteo 1" lot={lot} />
    </MemoryRouter>,
  )
}

describe('SellLotLink', () => {
  it('links to the sale page of the lote', () => {
    renderLink({ id: 'lot-1', number: '7', price: 120000, currency: 'USD' })

    expect(screen.getByRole('link', { name: 'Pasar a venta' })).toHaveAttribute(
      'href',
      '/ventas/nueva/loteo%201/lot-1',
    )
  })

  it('stays disabled and explains what the lote is missing', () => {
    renderLink({ id: 'lot-1', number: ' ', price: null, currency: '' })

    const button = screen.getByRole('button', { name: 'Pasar a venta' })
    expect(button).toBeDisabled()
    expect(screen.getByTitle('Completá el número y el precio del lote para habilitar la venta.')).toBeInTheDocument()
    expect(screen.queryByRole('link')).not.toBeInTheDocument()
  })
})
