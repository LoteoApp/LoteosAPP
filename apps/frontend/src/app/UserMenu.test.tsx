import type { ReactNode } from 'react'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { User } from '@supabase/supabase-js'
import UserMenu from './UserMenu'
import {
  AuthContext,
  type AuthContextValue,
} from '../features/auth/hooks/use-auth'

function renderUserMenu(
  auth: Partial<AuthContextValue> = {},
  children?: ReactNode,
) {
  const value: AuthContextValue = {
    isLoading: false,
    session: null,
    user: {
      email: 'leonel@loteosapp.com',
      user_metadata: { full_name: 'Leonel Zorzoli' },
    } as unknown as User,
    error: null,
    login: vi.fn(),
    logout: vi.fn().mockResolvedValue(undefined),
    ...auth,
  }

  render(
    <AuthContext.Provider value={value}>
      <UserMenu />
      {children}
    </AuthContext.Provider>,
  )

  return value
}

describe('UserMenu', () => {
  it('shows the account name and email next to the avatar', () => {
    renderUserMenu()

    expect(screen.getByText('Leonel Zorzoli')).toBeInTheDocument()
    expect(screen.getByText('leonel@loteosapp.com')).toBeInTheDocument()
  })

  it('opens the account menu when the trigger is clicked', async () => {
    const user = userEvent.setup()
    renderUserMenu()

    const trigger = screen.getByRole('button', {
      name: 'Cuenta de Leonel Zorzoli',
    })
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
    renderUserMenu()

    await user.click(
      screen.getByRole('button', { name: 'Cuenta de Leonel Zorzoli' }),
    )
    await user.click(screen.getByRole('menuitem', { name: 'Mi perfil' }))

    expect(
      screen.queryByRole('menuitem', { name: 'Mi perfil' }),
    ).not.toBeInTheDocument()
  })

  it('signs the user out and closes the menu when picking "Cerrar sesión"', async () => {
    const { logout } = renderUserMenu()
    const user = userEvent.setup()

    await user.click(
      screen.getByRole('button', { name: 'Cuenta de Leonel Zorzoli' }),
    )
    await user.click(screen.getByRole('menuitem', { name: 'Cerrar sesión' }))

    expect(logout).toHaveBeenCalledTimes(1)
    expect(
      screen.queryByRole('menuitem', { name: 'Cerrar sesión' }),
    ).not.toBeInTheDocument()
  })

  it('still closes the menu when signing out fails', async () => {
    renderUserMenu({
      logout: vi.fn().mockRejectedValue(new Error('sesión ya cerrada')),
    })
    const user = userEvent.setup()

    await user.click(
      screen.getByRole('button', { name: 'Cuenta de Leonel Zorzoli' }),
    )
    await user.click(screen.getByRole('menuitem', { name: 'Cerrar sesión' }))

    expect(
      screen.queryByRole('menuitem', { name: 'Cerrar sesión' }),
    ).not.toBeInTheDocument()
  })

  it('closes the menu when clicking outside of it', async () => {
    const user = userEvent.setup()
    renderUserMenu({}, <p>Fuera del menú</p>)

    await user.click(
      screen.getByRole('button', { name: 'Cuenta de Leonel Zorzoli' }),
    )
    await user.click(screen.getByText('Fuera del menú'))

    expect(
      screen.queryByRole('menuitem', { name: 'Mi perfil' }),
    ).not.toBeInTheDocument()
  })

  it('closes the menu when pressing Escape', async () => {
    const user = userEvent.setup()
    renderUserMenu()

    await user.click(
      screen.getByRole('button', { name: 'Cuenta de Leonel Zorzoli' }),
    )
    await user.keyboard('{Escape}')

    expect(
      screen.queryByRole('menuitem', { name: 'Mi perfil' }),
    ).not.toBeInTheDocument()
  })

  it('falls back to the email when the account has no full name', () => {
    renderUserMenu({ user: { email: 'leonel@loteosapp.com' } as User })

    expect(
      screen.getByRole('button', { name: 'Cuenta de leonel@loteosapp.com' }),
    ).toBeInTheDocument()
  })

  it('falls back to "Usuario" when the account has no name or email', () => {
    renderUserMenu({ user: {} as User })

    expect(
      screen.getByRole('button', { name: 'Cuenta de Usuario' }),
    ).toBeInTheDocument()
  })
})
