import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { createMemoryRouter, RouterProvider, useLocation } from 'react-router'
import type { Session } from '@supabase/supabase-js'
import RequireAuth from './RequireAuth'
import { AuthContext, type AuthContextValue } from '../hooks/use-auth'

function LoginScreen() {
  const location = useLocation()
  const from = (location.state as { from?: string } | null)?.from

  return <p>Inicio de sesión desde {from ?? 'ninguna ruta'}</p>
}

function renderProtectedRoute(auth: Partial<AuthContextValue>) {
  const value: AuthContextValue = {
    isLoading: false,
    session: null,
    user: null,
    error: null,
    login: vi.fn(),
    logout: vi.fn(),
    ...auth,
  }

  const router = createMemoryRouter(
    [
      {
        path: '/lotes',
        element: (
          <RequireAuth>
            <p>Contenido protegido</p>
          </RequireAuth>
        ),
      },
      { path: '/login', element: <LoginScreen /> },
    ],
    { initialEntries: ['/lotes'] },
  )

  return render(
    <AuthContext.Provider value={value}>
      <RouterProvider router={router} />
    </AuthContext.Provider>,
  )
}

describe('RequireAuth', () => {
  it('shows a loading message while the session is being verified', () => {
    renderProtectedRoute({ isLoading: true })

    expect(screen.getByText('Verificando sesión...')).toBeInTheDocument()
    expect(screen.queryByText('Contenido protegido')).not.toBeInTheDocument()
  })

  it('sends the visitor to the login page when there is no session', async () => {
    renderProtectedRoute({ session: null })

    expect(await screen.findByText(/Inicio de sesión/)).toBeInTheDocument()
    expect(screen.queryByText('Contenido protegido')).not.toBeInTheDocument()
  })

  it('remembers the requested route so the login page can return to it', async () => {
    renderProtectedRoute({ session: null })

    expect(
      await screen.findByText('Inicio de sesión desde /lotes'),
    ).toBeInTheDocument()
  })

  it('renders the protected content once there is a session', () => {
    renderProtectedRoute({ session: {} as Session })

    expect(screen.getByText('Contenido protegido')).toBeInTheDocument()
  })
})
