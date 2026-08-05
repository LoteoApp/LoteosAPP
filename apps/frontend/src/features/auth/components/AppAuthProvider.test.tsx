import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { AuthProvider } from 'react-oidc-context'
import AppAuthProvider from './AppAuthProvider'
import { oidcConfig } from '../config/oidc-config'

vi.mock('react-oidc-context', () => ({
  AuthProvider: vi.fn(({ children }) => children),
}))

const AuthProviderMock = vi.mocked(AuthProvider)

describe('AppAuthProvider', () => {
  it('configures react-oidc-context with the app Keycloak settings and renders its children', () => {
    render(
      <AppAuthProvider>
        <p>Hijo</p>
      </AppAuthProvider>,
    )

    expect(screen.getByText('Hijo')).toBeInTheDocument()
    expect(AuthProviderMock.mock.calls[0][0]).toEqual(
      expect.objectContaining({
        authority: oidcConfig.authority,
        client_id: oidcConfig.client_id,
      }),
    )
  })
})
