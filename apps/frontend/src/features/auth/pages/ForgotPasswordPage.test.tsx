import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { createMemoryRouter, RouterProvider } from 'react-router'
import ForgotPasswordPage from './ForgotPasswordPage'
import { requestPasswordReset } from '../api/auth'
import { ApiError } from '../../../shared/api/client'

vi.mock('../api/auth', () => ({
  requestPasswordReset: vi.fn(),
}))

function renderForgotPasswordPage() {
  const router = createMemoryRouter(
    [
      { path: '/olvide-contrasena', element: <ForgotPasswordPage /> },
      { path: '/login', element: <p>Iniciar sesión</p> },
    ],
    { initialEntries: ['/olvide-contrasena'] },
  )

  render(<RouterProvider router={router} />)
}

async function submitEmail(email: string) {
  await userEvent.type(screen.getByLabelText('Correo electrónico'), email)
  await userEvent.click(screen.getByRole('button', { name: 'Enviar link' }))
}

describe('ForgotPasswordPage', () => {
  it('shows a generic confirmation after requesting a reset', async () => {
    vi.mocked(requestPasswordReset).mockResolvedValue(undefined)

    renderForgotPasswordPage()
    await submitEmail('ana@example.com')

    expect(requestPasswordReset).toHaveBeenCalledWith('ana@example.com')
    expect(
      await screen.findByText(/te enviamos un mail con un link para restablecer tu/),
    ).toBeInTheDocument()
    expect(screen.queryByLabelText('Correo electrónico')).not.toBeInTheDocument()
  })

  it('shows the backend error without confirming anything was sent', async () => {
    vi.mocked(requestPasswordReset).mockRejectedValue(
      new ApiError(
        'Ya te enviamos un mail. Esperá un momento antes de pedir otro',
        'password_reset_rate_limited',
        429,
      ),
    )

    renderForgotPasswordPage()
    await submitEmail('ana@example.com')

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Ya te enviamos un mail. Esperá un momento antes de pedir otro',
    )
    expect(screen.getByLabelText('Correo electrónico')).toBeInTheDocument()
  })

  it('links back to the login page', () => {
    renderForgotPasswordPage()

    expect(screen.getByRole('link', { name: 'Volver a iniciar sesión' })).toHaveAttribute(
      'href',
      '/login',
    )
  })
})
