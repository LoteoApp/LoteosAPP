import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import AgenciesPage from './AgenciesPage'

describe('AgenciesPage', () => {
  it('renders the section heading', () => {
    render(<AgenciesPage />)

    expect(
      screen.getByRole('heading', { name: 'Inmobiliaria' }),
    ).toBeInTheDocument()
  })
})
