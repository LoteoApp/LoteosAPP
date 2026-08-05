import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import LotsPage from './LotsPage'

describe('LotsPage', () => {
  it('renders the section heading', () => {
    render(<LotsPage />)

    expect(screen.getByRole('heading', { name: 'Loteos' })).toBeInTheDocument()
  })
})
