import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import LotesList from './LotesList'
import type { LoteoLote } from '../types'

function lote(overrides: Partial<LoteoLote> = {}): LoteoLote {
  return {
    id: 'lt-1',
    manzanaId: 'mz-1',
    numero: '7',
    estado: 'disponible',
    precio: 150000,
    moneda: 'USD',
    superficie: 300,
    caracteristicas: '',
    poligono: [],
    ...overrides,
  }
}

const manzanaNumberById = new Map([['mz-1', '1']])

describe('LotesList', () => {
  it('shows the number, manzana, area, price and state of each lote', () => {
    render(
      <LotesList
        lotes={[lote(), lote({ id: 'lt-2', numero: '8', estado: 'vendido' })]}
        manzanaNumberById={manzanaNumberById}
        onSelectLote={vi.fn()}
      />,
    )

    const rows = within(screen.getByRole('list', { name: 'Lotes del loteo' })).getAllByRole(
      'listitem',
    )
    expect(rows).toHaveLength(2)
    expect(within(rows[0]).getByText('300 m²')).toBeInTheDocument()
    expect(within(rows[0]).getByText(/150\.000/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Lote 7 · Mz 1 · Disponible' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Lote 8 · Mz 1 · Vendido' })).toBeInTheDocument()
  })

  it('falls back when the lote has no number, manzana, area or price', () => {
    render(
      <LotesList
        lotes={[lote({ numero: '', manzanaId: 'mz-9', superficie: null, precio: null })]}
        manzanaNumberById={manzanaNumberById}
        onSelectLote={vi.fn()}
      />,
    )

    const row = screen.getByRole('button', { name: 'Lote · Disponible' })
    expect(within(row).getAllByText('—')).toHaveLength(2)
  })

  it('reports the selected lote and forwards the selection', async () => {
    const user = userEvent.setup()
    const onSelectLote = vi.fn()
    render(
      <LotesList
        lotes={[lote(), lote({ id: 'lt-2', numero: '8' })]}
        manzanaNumberById={manzanaNumberById}
        selectedLoteId="lt-2"
        onSelectLote={onSelectLote}
      />,
    )

    expect(screen.getByRole('button', { name: 'Lote 8 · Mz 1 · Disponible' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )

    await user.click(screen.getByRole('button', { name: 'Lote 7 · Mz 1 · Disponible' }))

    expect(onSelectLote).toHaveBeenCalledWith('lt-1')
  })

  it('disables the rows when there is nothing to select', () => {
    render(<LotesList lotes={[lote()]} manzanaNumberById={manzanaNumberById} />)

    expect(screen.getByRole('button', { name: 'Lote 7 · Mz 1 · Disponible' })).toBeDisabled()
  })

  it('tells the user when no lote matches', () => {
    render(<LotesList lotes={[]} manzanaNumberById={manzanaNumberById} onSelectLote={vi.fn()} />)

    expect(
      screen.getByText('No hay lotes que coincidan con la búsqueda.'),
    ).toBeInTheDocument()
  })
})
