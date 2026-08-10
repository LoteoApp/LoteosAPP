import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { useAuth } from 'react-oidc-context'
import UserMenu from './UserMenu'

vi.mock('react-oidc-context', () => ({
  useAuth: vi.fn(),
}))

const useAuthMock = vi.mocked(useAuth)

function mockAuthenticatedUser(overrides: Record<string, unknown> = {}) {
  const signoutRedirect = vi.fn()

  useAuthMock.mockReturnValue({
    signoutRedirect,
    user: {
      profile: {
        preferred_username: 'lzorzoli',
        name: 'Leonel Zorzoli',
        email: 'leonel@loteosapp.com',
      },
    },
    ...overrides,
  } as unknown as ReturnType<typeof useAuth>)

  return { signoutRedirect }
}

describe('UserMenu', () => {
  beforeEach(() => {
    mockAuthenticatedUser()
  })

  it('shows the account name and email next to the avatar', () => {
    render(<UserMenu />)

    expect(screen.getByText('lzorzoli')).toBeInTheDocument()
    expect(screen.getByText('leonel@loteosapp.com')).toBeInTheDocument()
  })

  it('opens the account menu when the trigger is clicked', async () => {
    const user = userEvent.setup()
    render(<UserMenu />)

    const trigger = screen.getByRole('button', { name: 'Cuenta de lzorzoli' })
    expect(trigger).toHaveAttribute('aria-expanded', 'false')

    await user.click(trigger)

    expect(trigger).toHaveAttribute('aria-expanded', 'true')
    expect(
      screen.getByRole('menuitem', { name: 'Mi perfil' }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('menuitem', { name: 'Cerrar sesión' }),
    ).toBeInTheDocument()
  })

  it('closes the menu when picking "Mi perfil"', async () => {
    const user = userEvent.setup()
    render(<UserMenu />)

    await user.click(screen.getByRole('button', { name: 'Cuenta de lzorzoli' }))
    await user.click(screen.getByRole('menuitem', { name: 'Mi perfil' }))

    expect(
      screen.queryByRole('menuitem', { name: 'Mi perfil' }),
    ).not.toBeInTheDocument()
  })

  it('signs the user out and closes the menu when picking "Cerrar sesión"', async () => {
    const { signoutRedirect } = mockAuthenticatedUser()
    const user = userEvent.setup()
    render(<UserMenu />)

    await user.click(screen.getByRole('button', { name: 'Cuenta de lzorzoli' }))
    await user.click(screen.getByRole('menuitem', { name: 'Cerrar sesión' }))

    expect(signoutRedirect).toHaveBeenCalledTimes(1)
    expect(
      screen.queryByRole('menuitem', { name: 'Cerrar sesión' }),
    ).not.toBeInTheDocument()
  })

  it('closes the menu when clicking outside of it', async () => {
    const user = userEvent.setup()
    render(
      <div>
        <UserMenu />
        <p>Fuera del menú</p>
      </div>,
    )

    await user.click(screen.getByRole('button', { name: 'Cuenta de lzorzoli' }))
    await user.click(screen.getByText('Fuera del menú'))

    expect(
      screen.queryByRole('menuitem', { name: 'Mi perfil' }),
    ).not.toBeInTheDocument()
  })

  it('closes the menu when pressing Escape', async () => {
    const user = userEvent.setup()
    render(<UserMenu />)

    await user.click(screen.getByRole('button', { name: 'Cuenta de lzorzoli' }))
    await user.keyboard('{Escape}')

    expect(
      screen.queryByRole('menuitem', { name: 'Mi perfil' }),
    ).not.toBeInTheDocument()
  })

  it('falls back to the profile name when there is no preferred username', () => {
    mockAuthenticatedUser({
      user: { profile: { name: 'Leonel Zorzoli', email: 'leonel@loteosapp.com' } },
    })
    render(<UserMenu />)

    expect(screen.getByText('Leonel Zorzoli')).toBeInTheDocument()
  })
})
