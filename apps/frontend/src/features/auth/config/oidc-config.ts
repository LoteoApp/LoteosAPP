import type { AuthProviderNoUserManagerProps } from 'react-oidc-context'
import { keycloakClientId, keycloakRealm, keycloakUrl } from '../../../shared/config/env'

export const oidcConfig: AuthProviderNoUserManagerProps = {
  authority: `${keycloakUrl}/realms/${keycloakRealm}`,
  client_id: keycloakClientId,
  redirect_uri: window.location.origin,
  post_logout_redirect_uri: window.location.origin,
  scope: 'openid profile email',
  automaticSilentRenew: true,
  onSigninCallback: () => {
    window.history.replaceState({}, document.title, window.location.pathname)
  },
}
