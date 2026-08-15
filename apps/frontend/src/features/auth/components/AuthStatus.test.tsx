import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router'
import type { User } from '@supabase/supabase-js'
import AuthStatus from './AuthStatus'
import { AuthContext, type AuthContextValue } from '../hooks/use-auth'

function renderAuthStatus(auth: Partial<AuthContextValue>) {
  const value: AuthContextValue = {
    isLoading: false,
    session: null,
    user: null,
    error: null,
    login: vi.fn(),
    logout: vi.fn().mockResolvedValue(undefined),
    ...auth,
  }

  render(
    <AuthContext.Provider value={value}>
      <MemoryRouter>
        <AuthStatus />
      </MemoryRouter>
    </AuthContext.Provider>,
  )

  return value
}

describe('AuthStatus', () => {
  it('shows a loading message while the session is being verified', () => {
    renderAuthStatus({ isLoading: true })

    expect(screen.getByText('Verificando sesión...')).toBeInTheDocument()
  })

  it('lets the user retry from the login page after a session error', () => {
    renderAuthStatus({ error: new Error('token inválido') })

    expect(screen.getByRole('alert')).toHaveTextContent('token inválido')
    expect(screen.getByRole('link', { name: 'Reintentar' })).toHaveAttribute(
      'href',
      '/login',
    )
  })

  it('shows the signed-in user and logs out on click', async () => {
    const { logout } = renderAuthStatus({
      user: { email: 'lzapata@loteosapp.com' } as User,
    })

    expect(screen.getByText('lzapata@loteosapp.com')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Cerrar sesión' }))

    expect(logout).toHaveBeenCalledOnce()
  })

  it('keeps the session visible when logging out fails', async () => {
    renderAuthStatus({
      user: { email: 'lzapata@loteosapp.com' } as User,
      logout: vi.fn().mockRejectedValue(new Error('sesión ya cerrada')),
    })

    await userEvent.click(screen.getByRole('button', { name: 'Cerrar sesión' }))

    expect(screen.getByText('lzapata@loteosapp.com')).toBeInTheDocument()
  })

  it('links to the login page when there is no session', () => {
    renderAuthStatus({ user: null })

    expect(screen.getByRole('link', { name: 'Iniciar sesión' })).toHaveAttribute(
      'href',
      '/login',
    )
  })
})
