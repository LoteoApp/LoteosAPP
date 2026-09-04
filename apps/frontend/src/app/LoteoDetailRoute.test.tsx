import type { Session } from '@supabase/supabase-js'
import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { AuthContext, type AuthContextValue } from '../features/auth/hooks/use-auth'
import LoteoDetailRoute from './LoteoDetailRoute'

vi.mock('../features/lots/pages/LoteoDetailPage', () => ({
  default: ({ accessToken, canEdit }: { accessToken: string | null; canEdit?: boolean }) => (
    <output data-can-edit={canEdit ? 'yes' : 'no'}>{accessToken ?? 'no-session'}</output>
  ),
}))

function renderRoute(token: string | null, role?: string) {
  const value: AuthContextValue = {
    isLoading: false,
    session: token
      ? ({
          access_token: token,
          user: role ? { app_metadata: { role } } : undefined,
        } as unknown as Session)
      : null,
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
    expect(screen.getByRole('status')).toHaveAttribute('data-can-edit', 'no')
  })

  it('allows editing for administrator and agrimensor roles', () => {
    const { rerender } = renderRoute('session-token', 'administrador')
    expect(screen.getByRole('status')).toHaveAttribute('data-can-edit', 'yes')

    rerender(
      <AuthContext.Provider
        value={{
          isLoading: false,
          session: {
            access_token: 'session-token',
            user: { app_metadata: { role: 'agrimensor' } },
          } as unknown as Session,
          user: null,
          error: null,
          login: vi.fn(),
          logout: vi.fn(),
        }}
      >
        <LoteoDetailRoute />
      </AuthContext.Provider>,
    )
    expect(screen.getByRole('status')).toHaveAttribute('data-can-edit', 'yes')
  })

  it('injects a null token when the session is unavailable', () => {
    renderRoute(null)

    expect(screen.getByText('no-session')).toBeInTheDocument()
  })
})
