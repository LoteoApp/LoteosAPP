import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { createMemoryRouter, RouterProvider } from 'react-router'
import type { Session } from '@supabase/supabase-js'
import LoginPage from './LoginPage'
import { AuthContext, type AuthContextValue } from '../hooks/use-auth'

function renderLoginPage(
  auth: Partial<AuthContextValue> = {},
  initialEntry: string | { pathname: string; state: unknown } = '/login',
) {
  const value: AuthContextValue = {
    isLoading: false,
    session: null,
    user: null,
    error: null,
    login: vi.fn().mockResolvedValue(undefined),
    logout: vi.fn(),
    ...auth,
  }

  const router = createMemoryRouter(
    [
      { path: '/login', element: <LoginPage /> },
      { path: '/lotes', element: <p>Listado de lotes</p> },
      { path: '/clientes', element: <p>Listado de clientes</p> },
    ],
    { initialEntries: [initialEntry] },
  )

  render(
    <AuthContext.Provider value={value}>
      <RouterProvider router={router} />
    </AuthContext.Provider>,
  )

  return value
}

async function fillCredentials(email: string, password: string) {
  await userEvent.type(screen.getByLabelText('Correo electrónico'), email)
  await userEvent.type(screen.getByLabelText('Contraseña'), password)
  await userEvent.click(screen.getByRole('button', { name: 'Ingresar' }))
}

describe('LoginPage', () => {
  it('shows a loading message while the session is being verified', () => {
    renderLoginPage({ isLoading: true })

    expect(screen.getByText('Verificando sesión...')).toBeInTheDocument()
    expect(screen.queryByLabelText('Correo electrónico')).not.toBeInTheDocument()
  })

  it('signs the user in and takes them to the lots section', async () => {
    const { login } = renderLoginPage()

    await fillCredentials('leonel@loteosapp.com', 'secreto')

    expect(login).toHaveBeenCalledWith('leonel@loteosapp.com', 'secreto')
    expect(await screen.findByText('Listado de lotes')).toBeInTheDocument()
  })

  it('returns to the route the visitor originally requested', async () => {
    renderLoginPage({}, { pathname: '/login', state: { from: '/clientes' } })

    await fillCredentials('leonel@loteosapp.com', 'secreto')

    expect(await screen.findByText('Listado de clientes')).toBeInTheDocument()
  })

  it('ignores a requested route that points at an external host', async () => {
    renderLoginPage({}, { pathname: '/login', state: { from: '//example.com' } })

    await fillCredentials('leonel@loteosapp.com', 'secreto')

    expect(await screen.findByText('Listado de lotes')).toBeInTheDocument()
  })

  it('shows the reason in Spanish when the credentials are rejected', async () => {
    renderLoginPage({
      login: vi.fn().mockRejectedValue(
        Object.assign(new Error('Invalid login credentials'), {
          code: 'invalid_credentials',
        }),
      ),
    })

    await fillCredentials('leonel@loteosapp.com', 'incorrecta')

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Correo o contraseña incorrectos.',
    )
    expect(screen.getByRole('button', { name: 'Ingresar' })).toBeEnabled()
  })

  it('skips the form when there is already a session', async () => {
    renderLoginPage({ session: {} as Session })

    expect(await screen.findByText('Listado de lotes')).toBeInTheDocument()
    expect(screen.queryByLabelText('Correo electrónico')).not.toBeInTheDocument()
  })
})
