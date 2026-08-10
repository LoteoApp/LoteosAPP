import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import ClientsPage from './ClientsPage'

describe('ClientsPage', () => {
  it('renders the section heading', () => {
    render(<ClientsPage />)

    expect(screen.getByRole('heading', { name: 'Clientes' })).toBeInTheDocument()
  })
})
