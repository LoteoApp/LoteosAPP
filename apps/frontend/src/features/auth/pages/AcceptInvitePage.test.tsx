import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createMemoryRouter, RouterProvider } from 'react-router'
import type { AuthError, User } from '@supabase/supabase-js'
import AcceptInvitePage from './AcceptInvitePage'
import { supabaseClient } from '../../../shared/config/supabase-client'

vi.mock('../../../shared/config/supabase-client', () => ({
  supabaseClient: {
    auth: {
      verifyOtp: vi.fn(),
      updateUser: vi.fn(),
      signOut: vi.fn(),
    },
  },
}))

const fakeUser = { id: 'user-1' } as unknown as User

function authError(code: string, message: string): AuthError {
  return Object.assign(new Error(message), { code }) as unknown as AuthError
}

function renderAcceptInvitePage(fragment = '') {
  window.history.pushState(null, '', '/aceptar-invitacion' + (fragment ? `#${fragment}` : ''))

  const router = createMemoryRouter(
    [
      { path: '/aceptar-invitacion', element: <AcceptInvitePage /> },
      { path: '/login', element: <p>Iniciar sesión</p> },
    ],
    { initialEntries: ['/aceptar-invitacion'] },
  )

  render(<RouterProvider router={router} />)
}

async function fillPasswords(password: string, confirmPassword: string) {
  await userEvent.type(screen.getByLabelText('Elegí tu contraseña'), password)
  await userEvent.type(screen.getByLabelText('Repetir contraseña'), confirmPassword)
  await userEvent.click(screen.getByRole('button', { name: 'Activar cuenta' }))
}

beforeEach(() => {
  vi.mocked(supabaseClient.auth.verifyOtp).mockReset()
  vi.mocked(supabaseClient.auth.updateUser).mockReset()
  vi.mocked(supabaseClient.auth.signOut).mockReset()
})

describe('AcceptInvitePage', () => {
  afterEach(() => {
    window.history.replaceState(null, '', '/')
  })

  it('shows an invalid link message when the url has no token_hash', () => {
    renderAcceptInvitePage()

    expect(screen.getByText('El link de invitación es inválido o venció.')).toBeInTheDocument()
    expect(screen.queryByLabelText('Elegí tu contraseña')).not.toBeInTheDocument()
  })

  it('clears the token from the url once it has been read', () => {
    renderAcceptInvitePage('token_hash=abc123&type=invite')

    expect(window.location.hash).toBe('')
  })

  it('rejects a password shorter than 8 characters without calling supabase', async () => {
    renderAcceptInvitePage('token_hash=abc123&type=invite')

    await fillPasswords('short', 'short')

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'La contraseña debe tener al menos 8 caracteres.',
    )
    expect(supabaseClient.auth.verifyOtp).not.toHaveBeenCalled()
  })

  it('rejects mismatched passwords without calling supabase', async () => {
    renderAcceptInvitePage('token_hash=abc123&type=invite')

    await fillPasswords('a-chosen-password', 'a-different-password')

    expect(await screen.findByRole('alert')).toHaveTextContent('Las contraseñas no coinciden.')
    expect(supabaseClient.auth.verifyOtp).not.toHaveBeenCalled()
  })

  it('verifies the token, sets the password and shows success', async () => {
    vi.mocked(supabaseClient.auth.verifyOtp).mockResolvedValue({
      data: { session: null, user: null },
      error: null,
    })
    vi.mocked(supabaseClient.auth.updateUser).mockResolvedValue({
      data: { user: fakeUser },
      error: null,
    })
    vi.mocked(supabaseClient.auth.signOut).mockResolvedValue({ error: null })

    renderAcceptInvitePage('token_hash=abc123&type=invite')
    await fillPasswords('a-chosen-password', 'a-chosen-password')

    expect(supabaseClient.auth.verifyOtp).toHaveBeenCalledWith({
      token_hash: 'abc123',
      type: 'invite',
    })
    expect(supabaseClient.auth.updateUser).toHaveBeenCalledWith({ password: 'a-chosen-password' })
    expect(await screen.findByText(/Tu cuenta quedó activada/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Iniciar sesión' })).toHaveAttribute('href', '/login')
    expect(supabaseClient.auth.signOut).toHaveBeenCalled()
  })

  it('shows a specific message when the token already expired or was used', async () => {
    vi.mocked(supabaseClient.auth.verifyOtp).mockResolvedValue({
      data: { session: null, user: null },
      error: authError('otp_expired', 'Token has expired or is invalid'),
    })

    renderAcceptInvitePage('token_hash=abc123&type=invite')
    await fillPasswords('a-chosen-password', 'a-chosen-password')

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'El link de invitación venció. Pedile a un administrador que te reenvíe uno nuevo.',
    )
    expect(supabaseClient.auth.updateUser).not.toHaveBeenCalled()
  })

  it('shows the backend error when updateUser fails', async () => {
    vi.mocked(supabaseClient.auth.verifyOtp).mockResolvedValue({
      data: { session: null, user: null },
      error: null,
    })
    vi.mocked(supabaseClient.auth.updateUser).mockResolvedValue({
      data: { user: null },
      error: authError('weak_password', 'Password should be at least 6 characters'),
    })

    renderAcceptInvitePage('token_hash=abc123&type=invite')
    await fillPasswords('a-chosen-password', 'a-chosen-password')

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'No se pudo activar la cuenta: Password should be at least 6 characters',
    )
  })

  it('retries only updateUser after the token was already verified', async () => {
    vi.mocked(supabaseClient.auth.verifyOtp).mockResolvedValue({
      data: { session: null, user: null },
      error: null,
    })
    vi.mocked(supabaseClient.auth.updateUser)
      .mockResolvedValueOnce({
        data: { user: null },
        error: authError('network_error', 'Failed to fetch'),
      })
      .mockResolvedValueOnce({ data: { user: fakeUser }, error: null })
    vi.mocked(supabaseClient.auth.signOut).mockResolvedValue({ error: null })

    renderAcceptInvitePage('token_hash=abc123&type=invite')
    await fillPasswords('a-chosen-password', 'a-chosen-password')
    expect(await screen.findByRole('alert')).toHaveTextContent('No se pudo activar la cuenta: Failed to fetch')

    await userEvent.click(screen.getByRole('button', { name: 'Activar cuenta' }))

    expect(await screen.findByText(/Tu cuenta quedó activada/)).toBeInTheDocument()
    expect(supabaseClient.auth.verifyOtp).toHaveBeenCalledTimes(1)
    expect(supabaseClient.auth.updateUser).toHaveBeenCalledTimes(2)
  })
})
