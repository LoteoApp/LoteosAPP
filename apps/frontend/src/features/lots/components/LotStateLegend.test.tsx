import { render, screen, within } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import LotStateLegend from './LotStateLegend'

describe('LotStateLegend', () => {
  it('lists every lot state by default', () => {
    render(<LotStateLegend />)

    const legend = screen.getByRole('list', { name: 'Colores por estado del lote' })
    expect(within(legend).getAllByRole('listitem')).toHaveLength(4)
    expect(within(legend).getByText('Disponible')).toBeInTheDocument()
    expect(within(legend).getByText('Finalizado')).toBeInTheDocument()
  })

  it('lists only the given states', () => {
    render(<LotStateLegend states={['reservado', 'vendido']} />)

    const legend = screen.getByRole('list', { name: 'Colores por estado del lote' })
    expect(within(legend).getByText('Reservado')).toBeInTheDocument()
    expect(within(legend).getByText('Vendido')).toBeInTheDocument()
    expect(within(legend).queryByText('Disponible')).not.toBeInTheDocument()
  })

  it('renders nothing without states', () => {
    render(<LotStateLegend states={[]} />)

    expect(
      screen.queryByRole('list', { name: 'Colores por estado del lote' }),
    ).not.toBeInTheDocument()
  })
})
