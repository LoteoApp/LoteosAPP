import { renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import type { ReactNode } from 'react'
import { AuthContext, useAuth, type AuthContextValue } from './use-auth'

describe('useAuth', () => {
  it('throws when used outside an AppAuthProvider', () => {
    expect(() => renderHook(() => useAuth())).toThrow(
      'useAuth must be used within an AppAuthProvider',
    )
  })

  it('returns the context value provided by AppAuthProvider', () => {
    const value: AuthContextValue = {
      isLoading: false,
      session: null,
      user: null,
      error: null,
      login: async () => {},
      logout: async () => {},
    }
    const wrapper = ({ children }: { children: ReactNode }) => (
      <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
    )

    const { result } = renderHook(() => useAuth(), { wrapper })

    expect(result.current).toBe(value)
  })
})
