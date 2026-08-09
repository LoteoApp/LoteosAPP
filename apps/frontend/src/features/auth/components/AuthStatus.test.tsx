import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { useAuth } from 'react-oidc-context'
import AuthStatus from './AuthStatus'

vi.mock('react-oidc-context', () => ({
  useAuth: vi.fn(),
}))

const useAuthMock = vi.mocked(useAuth)

describe('AuthStatus', () => {
  it('shows a loading message while the session is being verified', () => {
    useAuthMock.mockReturnValue({ isLoading: true } as ReturnType<typeof useAuth>)

    render(<AuthStatus />)

    expect(screen.getByText('Verificando sesión...')).toBeInTheDocument()
  })

  it('lets the user retry sign-in after a session error', async () => {
    const signinRedirect = vi.fn()
    useAuthMock.mockReturnValue({
      isLoading: false,
      error: new Error('token inválido'),
      signinRedirect,
    } as unknown as ReturnType<typeof useAuth>)

    render(<AuthStatus />)

    expect(screen.getByRole('alert')).toHaveTextContent('token inválido')
    await userEvent.click(screen.getByRole('button', { name: 'Reintentar' }))

    expect(signinRedirect).toHaveBeenCalledOnce()
  })

  it('shows the signed-in user and logs out on click', async () => {
    const signoutRedirect = vi.fn()
    useAuthMock.mockReturnValue({
      isLoading: false,
      error: undefined,
      isAuthenticated: true,
      user: { profile: { preferred_username: 'lzapata' } },
      signoutRedirect,
    } as unknown as ReturnType<typeof useAuth>)

    render(<AuthStatus />)

    expect(screen.getByText('lzapata')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Cerrar sesión' }))

    expect(signoutRedirect).toHaveBeenCalledOnce()
  })

  it('offers to sign in when there is no session', async () => {
    const signinRedirect = vi.fn()
    useAuthMock.mockReturnValue({
      isLoading: false,
      error: undefined,
      isAuthenticated: false,
      signinRedirect,
    } as unknown as ReturnType<typeof useAuth>)

    render(<AuthStatus />)

    await userEvent.click(screen.getByRole('button', { name: 'Iniciar sesión' }))

    expect(signinRedirect).toHaveBeenCalledOnce()
  })
})
