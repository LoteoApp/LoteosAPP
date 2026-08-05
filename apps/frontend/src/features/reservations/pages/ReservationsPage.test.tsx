import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import ReservationsPage from './ReservationsPage'

describe('ReservationsPage', () => {
  it('renders the section heading', () => {
    render(<ReservationsPage />)

    expect(screen.getByRole('heading', { name: 'Reservas' })).toBeInTheDocument()
  })
})
