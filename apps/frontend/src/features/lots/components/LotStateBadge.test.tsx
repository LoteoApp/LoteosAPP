import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import LotStateBadge from './LotStateBadge'
import type { LotState } from '../types'

describe('LotStateBadge', () => {
  it.each([
    ['disponible', 'Disponible'],
    ['reservado', 'Reservado'],
    ['vendido', 'Vendido'],
    ['finalizado', 'Finalizado'],
  ] satisfies [LotState, string][])('shows %s as %s', (state, label) => {
    render(<LotStateBadge state={state} />)

    expect(screen.getByText(label)).toBeInTheDocument()
  })
})
