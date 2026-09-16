import type { Session } from '@supabase/supabase-js'
import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { AuthContext, type AuthContextValue } from '../features/auth/hooks/use-auth'
import UsersRoute from './UsersRoute'

vi.mock('../features/users/pages/UsersPage', () => ({
  default: ({ accessToken }: { accessToken: string | null }) => (
    <output>{accessToken ?? 'no-session'}</output>
  ),
}))

function renderRoute(token: string | null) {
  const value: AuthContextValue = {
    isLoading: false,
    session: token ? ({ access_token: token } as unknown as Session) : null,
    user: null,
    error: null,
    login: vi.fn(),
    logout: vi.fn(),
  }

  return render(
    <AuthContext.Provider value={value}>
      <UsersRoute />
    </AuthContext.Provider>,
  )
}

describe('UsersRoute', () => {
  it('injects the current access token into the users feature', () => {
    renderRoute('session-token')

    expect(screen.getByText('session-token')).toBeInTheDocument()
  })

  it('injects a null token when the session is unavailable', () => {
    renderRoute(null)

    expect(screen.getByText('no-session')).toBeInTheDocument()
  })
})
