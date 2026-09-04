import type { Session } from '@supabase/supabase-js'
import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { AuthContext, type AuthContextValue } from '../features/auth/hooks/use-auth'
import LoteoDetailRoute from './LoteoDetailRoute'

vi.mock('../features/lots/pages/LoteoDetailPage', () => ({
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
      <LoteoDetailRoute />
    </AuthContext.Provider>,
  )
}

describe('LoteoDetailRoute', () => {
  it('injects the current access token into the loteo detail page', () => {
    renderRoute('session-token')

    expect(screen.getByText('session-token')).toBeInTheDocument()
  })

  it('injects a null token when the session is unavailable', () => {
    renderRoute(null)

    expect(screen.getByText('no-session')).toBeInTheDocument()
  })
})
