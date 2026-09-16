import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { createMemoryRouter, RouterProvider } from 'react-router'
import type { User } from '@supabase/supabase-js'
import RequireRole from './RequireRole'
import { AuthContext, type AuthContextValue } from '../hooks/use-auth'

function userWithRole(role: string | null): User | null {
  return role ? ({ app_metadata: { role } } as unknown as User) : null
}

function renderGuardedRoute(role: string | null, allowedRoles: readonly string[]) {
  const value: AuthContextValue = {
    isLoading: false,
    session: null,
    user: userWithRole(role),
    error: null,
    login: vi.fn(),
    logout: vi.fn(),
  }

  const router = createMemoryRouter(
    [
      {
        path: '/usuarios',
        element: (
          <RequireRole roles={allowedRoles}>
            <p>Contenido protegido</p>
          </RequireRole>
        ),
      },
      { path: '/lotes', element: <p>Contenido de lotes</p> },
    ],
    { initialEntries: ['/usuarios'] },
  )

  return render(
    <AuthContext.Provider value={value}>
      <RouterProvider router={router} />
    </AuthContext.Provider>,
  )
}

describe('RequireRole', () => {
  it('renders the protected content when the role is allowed', () => {
    renderGuardedRoute('administrador', ['administrador'])

    expect(screen.getByText('Contenido protegido')).toBeInTheDocument()
  })

  it('renders the protected content when the role is one of several allowed', () => {
    renderGuardedRoute('agrimensor', ['administrador', 'agrimensor'])

    expect(screen.getByText('Contenido protegido')).toBeInTheDocument()
  })

  it('sends a role outside the allowed list to /lotes', async () => {
    renderGuardedRoute('escribano', ['administrador', 'agrimensor'])

    expect(await screen.findByText('Contenido de lotes')).toBeInTheDocument()
    expect(screen.queryByText('Contenido protegido')).not.toBeInTheDocument()
  })

  it('sends a caller with no role to /lotes', async () => {
    renderGuardedRoute(null, ['administrador'])

    expect(await screen.findByText('Contenido de lotes')).toBeInTheDocument()
  })
})
