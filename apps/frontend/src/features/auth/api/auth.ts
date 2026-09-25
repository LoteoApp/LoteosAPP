import { apiFetch } from '../../../shared/api/client'

const AUTH_PATH = '/api/v1/auth'

export async function requestPasswordReset(email: string): Promise<void> {
  await apiFetch(`${AUTH_PATH}/recuperar-contrasena`, { method: 'POST', body: { email } })
}

export async function confirmPasswordReset(token: string, newPassword: string): Promise<void> {
  await apiFetch(`${AUTH_PATH}/restablecer-contrasena`, {
    method: 'POST',
    body: { token, newPassword },
  })
}
