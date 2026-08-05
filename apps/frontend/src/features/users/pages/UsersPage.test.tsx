import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import UsersPage from './UsersPage'

describe('UsersPage', () => {
  it('renders the section heading', () => {
    render(<UsersPage />)

    expect(screen.getByRole('heading', { name: 'Usuarios' })).toBeInTheDocument()
  })
})
