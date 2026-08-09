import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { useAuth } from 'react-oidc-context'
import RequireAuth from './RequireAuth'

vi.mock('react-oidc-context', () => ({
  useAuth: vi.fn(),
}))

const useAuthMock = vi.mocked(useAuth)

describe('RequireAuth', () => {
  it('shows a loading message while the session is being verified', () => {
    useAuthMock.mockReturnValue({
      isLoading: true,
      isAuthenticated: false,
    } as unknown as ReturnType<typeof useAuth>)

    render(
      <RequireAuth>
        <p>Contenido protegido</p>
      </RequireAuth>,
    )

    expect(screen.getByText('Verificando sesión...')).toBeInTheDocument()
    expect(screen.queryByText('Contenido protegido')).not.toBeInTheDocument()
  })

  it('redirects to sign-in when there is no session', () => {
    const signinRedirect = vi.fn()
    useAuthMock.mockReturnValue({
      isLoading: false,
      isAuthenticated: false,
      error: undefined,
      signinRedirect,
    } as unknown as ReturnType<typeof useAuth>)

    render(
      <RequireAuth>
        <p>Contenido protegido</p>
      </RequireAuth>,
    )

    expect(signinRedirect).toHaveBeenCalledOnce()
    expect(screen.getByText('Redirigiendo al inicio de sesión...')).toBeInTheDocument()
  })

  it('renders the protected content once authenticated', () => {
    useAuthMock.mockReturnValue({
      isLoading: false,
      isAuthenticated: true,
      error: undefined,
    } as unknown as ReturnType<typeof useAuth>)

    render(
      <RequireAuth>
        <p>Contenido protegido</p>
      </RequireAuth>,
    )

    expect(screen.getByText('Contenido protegido')).toBeInTheDocument()
  })

  it('shows an error instead of redirecting when the session check fails', () => {
    const signinRedirect = vi.fn()
    useAuthMock.mockReturnValue({
      isLoading: false,
      isAuthenticated: false,
      error: new Error('jwks no disponible'),
      signinRedirect,
    } as unknown as ReturnType<typeof useAuth>)

    render(
      <RequireAuth>
        <p>Contenido protegido</p>
      </RequireAuth>,
    )

    expect(screen.getByRole('alert')).toHaveTextContent('jwks no disponible')
    expect(signinRedirect).not.toHaveBeenCalled()
  })
})
