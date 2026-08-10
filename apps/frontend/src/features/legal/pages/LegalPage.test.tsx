import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import LegalPage from './LegalPage'

describe('LegalPage', () => {
  it('renders the section heading', () => {
    render(<LegalPage />)

    expect(
      screen.getByRole('heading', { name: 'Documentación legal' }),
    ).toBeInTheDocument()
  })
})
