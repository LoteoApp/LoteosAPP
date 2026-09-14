import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { describe, expect, it } from 'vitest'
import SellLotLink from './SellLotLink'

function renderLink(lote: { id: string; numero: string; precio: number | null }) {
  render(
    <MemoryRouter>
      <SellLotLink loteoId="loteo 1" lote={lote} />
    </MemoryRouter>,
  )
}

describe('SellLotLink', () => {
  it('links to the sale page of the lote', () => {
    renderLink({ id: 'lot-1', numero: '7', precio: 120000 })

    expect(screen.getByRole('link', { name: 'Pasar a venta' })).toHaveAttribute(
      'href',
      '/ventas/nueva/loteo%201/lot-1',
    )
  })

  it('stays disabled and explains what the lote is missing', () => {
    renderLink({ id: 'lot-1', numero: ' ', precio: null })

    const button = screen.getByRole('button', { name: 'Pasar a venta' })
    expect(button).toBeDisabled()
    expect(screen.getByTitle('Completá el número y el precio del lote para habilitar la venta.')).toBeInTheDocument()
    expect(screen.queryByRole('link')).not.toBeInTheDocument()
  })
})
