import { describe, expect, it, vi } from 'vitest'
import { oidcConfig } from './oidc-config'

describe('oidcConfig', () => {
  it('builds the authority from the Keycloak URL and realm', () => {
    expect(oidcConfig.authority).toBe('http://localhost:8081/realms/loteosapp')
    expect(oidcConfig.client_id).toBe('loteosapp-frontend')
    expect(oidcConfig.redirect_uri).toBe(window.location.origin)
  })

  it('strips the OIDC query params from the URL after sign-in', () => {
    const replaceStateSpy = vi.spyOn(window.history, 'replaceState')

    oidcConfig.onSigninCallback?.({} as never)

    expect(replaceStateSpy).toHaveBeenCalledWith({}, document.title, window.location.pathname)
  })
})
