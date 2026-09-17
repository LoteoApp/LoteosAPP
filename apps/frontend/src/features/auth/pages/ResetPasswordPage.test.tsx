import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { createMemoryRouter, RouterProvider } from 'react-router'
import ResetPasswordPage from './ResetPasswordPage'
import { confirmPasswordReset } from '../api/auth'
import { ApiError } from '../../../shared/api/client'

vi.mock('../api/auth', () => ({
  confirmPasswordReset: vi.fn(),
}))

function renderResetPasswordPage(initialEntry: string) {
  const router = createMemoryRouter(
    [
      { path: '/restablecer-contrasena', element: <ResetPasswordPage /> },
      { path: '/login', element: <p>Iniciar sesión</p> },
      { path: '/olvide-contrasena', element: <p>Olvidé mi contraseña</p> },
    ],
    { initialEntries: [initialEntry] },
  )

  render(<RouterProvider router={router} />)
}

async function fillPasswords(password: string, confirmPassword: string) {
  await userEvent.type(screen.getByLabelText('Contraseña nueva'), password)
  await userEvent.type(screen.getByLabelText('Repetir contraseña'), confirmPassword)
  await userEvent.click(screen.getByRole('button', { name: 'Guardar contraseña' }))
}

describe('ResetPasswordPage', () => {
  it('shows an invalid link message when the url has no token', () => {
    renderResetPasswordPage('/restablecer-contrasena')

    expect(
      screen.getByText('El link para restablecer la contraseña es inválido o venció.'),
    ).toBeInTheDocument()
    expect(screen.queryByLabelText('Contraseña nueva')).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Pedir un link nuevo' })).toHaveAttribute(
      'href',
      '/olvide-contrasena',
    )
  })

  it('rejects a password shorter than 8 characters without calling the backend', async () => {
    renderResetPasswordPage('/restablecer-contrasena?token=abc123')

    await fillPasswords('short', 'short')

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'La contraseña debe tener al menos 8 caracteres.',
    )
    expect(confirmPasswordReset).not.toHaveBeenCalled()
  })

  it('rejects mismatched passwords without calling the backend', async () => {
    renderResetPasswordPage('/restablecer-contrasena?token=abc123')

    await fillPasswords('a-new-password', 'a-different-password')

    expect(await screen.findByRole('alert')).toHaveTextContent('Las contraseñas no coinciden.')
    expect(confirmPasswordReset).not.toHaveBeenCalled()
  })

  it('confirms the reset with the token from the url and shows success', async () => {
    vi.mocked(confirmPasswordReset).mockResolvedValue(undefined)

    renderResetPasswordPage('/restablecer-contrasena?token=abc123')
    await fillPasswords('a-new-password', 'a-new-password')

    expect(confirmPasswordReset).toHaveBeenCalledWith('abc123', 'a-new-password')
    expect(await screen.findByText(/Tu contraseña se actualizó/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Iniciar sesión' })).toHaveAttribute('href', '/login')
  })

  it('shows the backend error for an invalid or expired token', async () => {
    vi.mocked(confirmPasswordReset).mockRejectedValue(
      new ApiError(
        'El link para restablecer la contraseña es inválido o venció',
        'password_reset_token_invalid',
        400,
      ),
    )

    renderResetPasswordPage('/restablecer-contrasena?token=abc123')
    await fillPasswords('a-new-password', 'a-new-password')

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'El link para restablecer la contraseña es inválido o venció',
    )
  })
})
