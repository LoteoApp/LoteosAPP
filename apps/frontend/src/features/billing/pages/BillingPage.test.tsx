import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import BillingPage from './BillingPage'

describe('BillingPage', () => {
  it('renders the section heading', () => {
    render(<BillingPage />)

    expect(screen.getByRole('heading', { name: 'Cobranzas' })).toBeInTheDocument()
  })
})
