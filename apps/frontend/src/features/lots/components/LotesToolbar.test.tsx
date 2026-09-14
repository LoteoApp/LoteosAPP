import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import LotesToolbar from './LotesToolbar'
import type { LotState } from '../types'

const counts: Record<LotState, number> = {
  disponible: 10,
  reservado: 5,
  vendido: 3,
  finalizado: 2,
}

describe('LotesToolbar', () => {
  it('reports what the user types in the search box', async () => {
    const user = userEvent.setup()
    const onSearchChange = vi.fn()
    render(
      <LotesToolbar
        search=""
        onSearchChange={onSearchChange}
        states={new Set()}
        onStatesChange={vi.fn()}
        counts={counts}
      />,
    )

    await user.type(screen.getByRole('searchbox', { name: 'Buscar lote o manzana' }), 'A')

    expect(onSearchChange).toHaveBeenCalledWith('A')
  })

  it('shows how many lotes are in each state', () => {
    render(
      <LotesToolbar
        search=""
        onSearchChange={vi.fn()}
        states={new Set()}
        onStatesChange={vi.fn()}
        counts={counts}
      />,
    )

    const filters = within(screen.getByRole('group', { name: 'Filtrar por estado' }))
    expect(filters.getByRole('button', { name: 'Disponible' })).toHaveTextContent('10')
    expect(filters.getByRole('button', { name: 'Finalizado' })).toHaveTextContent('2')
  })

  it('toggles a state filter on and off', async () => {
    const user = userEvent.setup()
    const onStatesChange = vi.fn()
    const { rerender } = render(
      <LotesToolbar
        search=""
        onSearchChange={vi.fn()}
        states={new Set()}
        onStatesChange={onStatesChange}
        counts={counts}
      />,
    )

    await user.click(screen.getByRole('button', { name: 'Reservado' }))
    expect(onStatesChange).toHaveBeenCalledWith(new Set(['reservado']))

    rerender(
      <LotesToolbar
        search=""
        onSearchChange={vi.fn()}
        states={new Set<LotState>(['reservado'])}
        onStatesChange={onStatesChange}
        counts={counts}
      />,
    )

    expect(screen.getByRole('button', { name: 'Reservado' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )

    await user.click(screen.getByRole('button', { name: 'Reservado' }))
    expect(onStatesChange).toHaveBeenLastCalledWith(new Set())
  })
})
