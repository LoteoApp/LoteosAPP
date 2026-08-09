import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import UserMenu from './UserMenu'

describe('UserMenu', () => {
  it('shows the account name and role next to the avatar', () => {
    render(<UserMenu />)

    expect(screen.getByText('Admin User')).toBeInTheDocument()
    expect(screen.getByText('Administrador')).toBeInTheDocument()
  })

  it('opens the account menu when the trigger is clicked', async () => {
    const user = userEvent.setup()
    render(<UserMenu />)

    const trigger = screen.getByRole('button', { name: 'Cuenta de Admin User' })
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

  it('closes the menu when picking an item', async () => {
    const user = userEvent.setup()
    render(<UserMenu />)

    await user.click(screen.getByRole('button', { name: 'Cuenta de Admin User' }))
    await user.click(screen.getByRole('menuitem', { name: 'Mi perfil' }))

    expect(
      screen.queryByRole('menuitem', { name: 'Mi perfil' }),
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

    await user.click(screen.getByRole('button', { name: 'Cuenta de Admin User' }))
    await user.click(screen.getByText('Fuera del menú'))

    expect(
      screen.queryByRole('menuitem', { name: 'Mi perfil' }),
    ).not.toBeInTheDocument()
  })

  it('closes the menu when pressing Escape', async () => {
    const user = userEvent.setup()
    render(<UserMenu />)

    await user.click(screen.getByRole('button', { name: 'Cuenta de Admin User' }))
    await user.keyboard('{Escape}')

    expect(
      screen.queryByRole('menuitem', { name: 'Mi perfil' }),
    ).not.toBeInTheDocument()
  })
})
