import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import SalesPage from './SalesPage'

describe('SalesPage', () => {
  it('renders the section heading', () => {
    render(<SalesPage />)

    expect(screen.getByRole('heading', { name: 'Ventas' })).toBeInTheDocument()
  })
})
