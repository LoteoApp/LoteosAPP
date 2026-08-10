export const apiUrl = (import.meta.env.VITE_API_URL ?? 'http://localhost:8080').replace(
  /\/$/,
  '',
)

export const keycloakUrl = (
  import.meta.env.VITE_KEYCLOAK_URL ?? 'http://localhost:8081'
).replace(/\/$/, '')
export const keycloakRealm = import.meta.env.VITE_KEYCLOAK_REALM ?? 'loteosapp'
export const keycloakClientId = import.meta.env.VITE_KEYCLOAK_CLIENT_ID ?? 'loteosapp-frontend'
