import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import LotesTable from './LotesTable'
import type { LoteoLote } from '../types'

function lote(overrides: Partial<LoteoLote> = {}): LoteoLote {
  return {
    id: 'lt-1',
    manzanaId: 'mz-1',
    numero: '7',
    precio: 150000,
    moneda: 'USD',
    superficie: 300,
    caracteristicas: 'Esquina',
    poligono: [],
    ...overrides,
  }
}

const manzanaNumberById = new Map([
  ['mz-1', '1'],
  ['mz-2', '2'],
])

describe('LotesTable', () => {
  it('renders a row per lote with its manzana number, area and price', () => {
    render(
      <LotesTable
        lotes={[lote(), lote({ id: 'lt-2', numero: '8', manzanaId: 'mz-2', precio: 90000 })]}
        manzanaNumberById={manzanaNumberById}
      />,
    )

    const rows = screen.getAllByRole('row')
    expect(rows).toHaveLength(3)

    const first = within(rows[1])
    expect(first.getByText('1')).toBeInTheDocument()
    expect(first.getByText('7')).toBeInTheDocument()
    expect(first.getByText('300 m²')).toBeInTheDocument()
    expect(first.getByText(/150\.000/)).toBeInTheDocument()
  })

  it('shows a dash where a lote has no price, surface, number or features', () => {
    render(
      <LotesTable
        lotes={[
          lote({ precio: null, superficie: null, numero: '', caracteristicas: '' }),
        ]}
        manzanaNumberById={manzanaNumberById}
      />,
    )

    const [, loteCell, superficieCell, precioCell, caracteristicasCell] = within(
      screen.getAllByRole('row')[1],
    ).getAllByRole('cell')
    expect(loteCell).toHaveTextContent('—')
    expect(superficieCell).toHaveTextContent('—')
    expect(precioCell).toHaveTextContent('—')
    expect(caracteristicasCell).toHaveTextContent('—')
  })

  it('shows a dash for a lote whose manzana is unknown', () => {
    render(
      <LotesTable lotes={[lote({ manzanaId: 'ghost' })]} manzanaNumberById={manzanaNumberById} />,
    )

    const cells = within(screen.getAllByRole('row')[1]).getAllByRole('cell')
    expect(cells[0]).toHaveTextContent('—')
  })

  it('shows an empty message when there are no lotes', () => {
    render(<LotesTable lotes={[]} manzanaNumberById={manzanaNumberById} />)

    expect(screen.getByText('No hay lotes para mostrar.')).toBeInTheDocument()
  })

  it('captions the table with the lote count', () => {
    render(<LotesTable lotes={[lote(), lote({ id: 'lt-2' })]} manzanaNumberById={manzanaNumberById} />)

    expect(screen.getByText('2 lotes')).toBeInTheDocument()
  })

  it('selects a lote from a row click and from the keyboard', async () => {
    const user = userEvent.setup()
    const onSelectLote = vi.fn()

    render(
      <LotesTable
        lotes={[lote(), lote({ id: 'lt-2', numero: '8' })]}
        manzanaNumberById={manzanaNumberById}
        selectedLoteId="lt-1"
        onSelectLote={onSelectLote}
      />,
    )

    expect(screen.getByRole('row', { name: 'Lote 7' })).toHaveAttribute('aria-selected', 'true')

    await user.click(screen.getByRole('row', { name: 'Lote 8' }))
    expect(onSelectLote).toHaveBeenCalledWith('lt-2')

    screen.getByRole('row', { name: 'Lote 7' }).focus()
    await user.keyboard('{Enter}')
    expect(onSelectLote).toHaveBeenCalledWith('lt-1')

    onSelectLote.mockClear()
    await user.keyboard(' ')
    expect(onSelectLote).toHaveBeenCalledWith('lt-1')

    onSelectLote.mockClear()
    await user.keyboard('a')
    expect(onSelectLote).not.toHaveBeenCalled()
  })
})
