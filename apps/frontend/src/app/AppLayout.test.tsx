import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { describe, expect, it, vi } from 'vitest'
import type { User } from '@supabase/supabase-js'
import AppLayout from './AppLayout'
import {
  AuthContext,
  type AuthContextValue,
} from '../features/auth/hooks/use-auth'

const authValue: AuthContextValue = {
  isLoading: false,
  session: null,
  user: {
    email: 'leonel@loteosapp.com',
    user_metadata: { full_name: 'Leonel Zorzoli' },
    app_metadata: { role: 'administrador' },
  } as unknown as User,
  error: null,
  login: vi.fn(),
  logout: vi.fn().mockResolvedValue(undefined),
}

const sectionLabels = [
  'Lotes',
  'Clientes',
  'Reservas',
  'Ventas',
  'Cobranzas',
  'Usuarios',
  'Inmobiliaria',
]

function stubDesktopMatchMedia(initialMatches = true) {
  const listeners = new Set<(event: { matches: boolean }) => void>()
  const mediaQueryList = {
    matches: initialMatches,
    addEventListener: (
      _event: string,
      handler: (event: { matches: boolean }) => void,
    ) => {
      listeners.add(handler)
    },
    removeEventListener: (
      _event: string,
      handler: (event: { matches: boolean }) => void,
    ) => {
      listeners.delete(handler)
    },
  }

  vi.stubGlobal('matchMedia', vi.fn().mockReturnValue(mediaQueryList))

  return {
    emitChange(matches: boolean) {
      for (const listener of listeners) {
        listener({ matches })
      }
    },
  }
}

function renderLayoutAt(path: string, role: string | null = 'administrador') {
  const router = createMemoryRouter(
    [
      {
        element: <AppLayout />,
        children: [
          { path: '/lotes', element: <p>Contenido de lotes</p> },
          { path: '/clientes', element: <p>Contenido de clientes</p> },
        ],
      },
    ],
    { initialEntries: [path] },
  )

  const value: AuthContextValue = {
    ...authValue,
    user: {
      ...authValue.user,
      app_metadata: { role },
    } as unknown as User,
  }

  return render(
    <AuthContext.Provider value={value}>
      <RouterProvider router={router} />
    </AuthContext.Provider>,
  )
}

describe('AppLayout', () => {
  it('renders a tab for every section', () => {
    renderLayoutAt('/lotes')

    for (const label of sectionLabels) {
      expect(screen.getByRole('link', { name: label })).toBeInTheDocument()
    }
  })

  it('marks the current section as the active tab', () => {
    renderLayoutAt('/clientes')

    expect(
      screen.getByRole('link', { name: 'Clientes', current: 'page' }),
    ).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Lotes' })).not.toHaveAttribute(
      'aria-current',
    )
  })

  it('renders the matched route content in the content area', () => {
    renderLayoutAt('/lotes')

    expect(screen.getByText('Contenido de lotes')).toBeInTheDocument()
  })

  it('links the app name to the clients section', () => {
    renderLayoutAt('/lotes')

    expect(screen.getByRole('link', { name: 'LoteosAPP' })).toHaveAttribute(
      'href',
      '/clientes',
    )
  })

  it('toggles the mobile menu button when opened and closed', async () => {
    const user = userEvent.setup()
    renderLayoutAt('/lotes')

    const toggle = screen.getByRole('button', { name: 'Abrir menú' })
    expect(toggle).toHaveAttribute('aria-expanded', 'false')

    await user.click(toggle)
    expect(
      screen.getByRole('button', { name: 'Cerrar menú' }),
    ).toHaveAttribute('aria-expanded', 'true')

    await user.click(screen.getByRole('button', { name: 'Cerrar menú' }))
    expect(
      screen.getByRole('button', { name: 'Abrir menú' }),
    ).toHaveAttribute('aria-expanded', 'false')
  })

  it('closes the mobile menu after picking a section', async () => {
    const user = userEvent.setup()
    renderLayoutAt('/lotes')

    await user.click(screen.getByRole('button', { name: 'Abrir menú' }))
    await user.click(screen.getByRole('link', { name: 'Clientes' }))

    expect(screen.getByRole('button', { name: 'Abrir menú' })).toHaveAttribute(
      'aria-expanded',
      'false',
    )
    expect(screen.getByText('Contenido de clientes')).toBeInTheDocument()
  })

  it('closes the mobile menu when the backdrop is clicked', async () => {
    const user = userEvent.setup()
    renderLayoutAt('/lotes')

    await user.click(screen.getByRole('button', { name: 'Abrir menú' }))
    await user.click(screen.getByRole('button', { name: 'Cerrar menú lateral' }))

    expect(screen.getByRole('button', { name: 'Abrir menú' })).toHaveAttribute(
      'aria-expanded',
      'false',
    )
  })

  it('opens the sidebar by default on desktop-sized viewports', () => {
    stubDesktopMatchMedia()

    renderLayoutAt('/lotes')

    expect(screen.getByRole('button', { name: 'Cerrar menú' })).toHaveAttribute(
      'aria-expanded',
      'true',
    )
  })

  it('keeps the sidebar open after picking a section on desktop-sized viewports', async () => {
    const user = userEvent.setup()
    stubDesktopMatchMedia()

    renderLayoutAt('/lotes')

    await user.click(screen.getByRole('link', { name: 'Clientes' }))

    expect(screen.getByRole('button', { name: 'Cerrar menú' })).toHaveAttribute(
      'aria-expanded',
      'true',
    )
    expect(screen.getByText('Contenido de clientes')).toBeInTheDocument()
  })

  it('syncs the sidebar when the viewport crosses the desktop breakpoint', () => {
    const { emitChange } = stubDesktopMatchMedia(true)

    renderLayoutAt('/lotes')

    expect(screen.getByRole('button', { name: 'Cerrar menú' })).toHaveAttribute(
      'aria-expanded',
      'true',
    )

    act(() => {
      emitChange(false)
    })

    expect(screen.getByRole('button', { name: 'Abrir menú' })).toHaveAttribute(
      'aria-expanded',
      'false',
    )

    act(() => {
      emitChange(true)
    })

    expect(screen.getByRole('button', { name: 'Cerrar menú' })).toHaveAttribute(
      'aria-expanded',
      'true',
    )
  })

  it('hides the Usuarios section for a non-administrador role', () => {
    renderLayoutAt('/lotes', 'administrativo')

    expect(screen.queryByRole('link', { name: 'Usuarios' })).not.toBeInTheDocument()
    for (const label of sectionLabels.filter((section) => section !== 'Usuarios')) {
      expect(screen.getByRole('link', { name: label })).toBeInTheDocument()
    }
  })

  it('hides the Usuarios section when the caller has no role', () => {
    renderLayoutAt('/lotes', null)

    expect(screen.queryByRole('link', { name: 'Usuarios' })).not.toBeInTheDocument()
  })
})
