import type { Session, User } from '@supabase/supabase-js'
import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { AuthContext, type AuthContextValue } from '../features/auth/hooks/use-auth'
import AgenciesRoute from './AgenciesRoute'

vi.mock('../features/agencies/pages/AgenciesPage', () => ({
  default: ({
    accessToken,
    isAdmin,
  }: {
    accessToken: string | null
    isAdmin: boolean
  }) => (
    <output>
      {accessToken ?? 'no-session'} | {isAdmin ? 'administrador' : 'otro rol'}
    </output>
  ),
}))

function renderRoute(token: string | null, role: string | null) {
  const value: AuthContextValue = {
    isLoading: false,
    session: token ? ({ access_token: token } as unknown as Session) : null,
    user: role ? ({ app_metadata: { role } } as unknown as User) : null,
    error: null,
    login: vi.fn(),
    logout: vi.fn(),
  }

  return render(
    <AuthContext.Provider value={value}>
      <AgenciesRoute />
    </AuthContext.Provider>,
  )
}

describe('AgenciesRoute', () => {
  it('injects the current access token and the administrador role', () => {
    renderRoute('session-token', 'administrador')

    expect(screen.getByText(/session-token \| administrador/)).toBeInTheDocument()
  })

  it('reports a caller of another role as not administrador', () => {
    renderRoute('session-token', 'administrativo')

    expect(screen.getByText(/session-token \| otro rol/)).toBeInTheDocument()
  })

  it('injects a null token when the session is unavailable', () => {
    renderRoute(null, null)

    expect(screen.getByText(/no-session \| otro rol/)).toBeInTheDocument()
  })
})
