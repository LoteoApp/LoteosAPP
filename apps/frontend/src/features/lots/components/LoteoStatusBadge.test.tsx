import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import LoteoStatusBadge from './LoteoStatusBadge'

describe('LoteoStatusBadge', () => {
  it('shows a pending badge when there is no plan', () => {
    render(<LoteoStatusBadge hasPlan={false} />)

    expect(screen.getByText('Plano pendiente')).toBeInTheDocument()
  })

  it('shows a loaded badge when there is a plan', () => {
    render(<LoteoStatusBadge hasPlan />)

    expect(screen.getByText('Plano cargado')).toBeInTheDocument()
  })
})
