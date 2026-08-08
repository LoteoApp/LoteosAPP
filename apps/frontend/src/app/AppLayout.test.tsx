import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { describe, expect, it } from 'vitest'
import AppLayout from './AppLayout'

const sectionLabels = [
  'Lotes',
  'Clientes',
  'Reservas',
  'Ventas',
  'Cobranzas',
  'Usuarios',
  'Documentación',
]

function renderLayoutAt(path: string) {
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

  return render(<RouterProvider router={router} />)
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

    expect(
      screen.getByRole('button', { name: 'Abrir menú' }),
    ).toHaveAttribute('aria-expanded', 'false')
    expect(screen.getByText('Contenido de clientes')).toBeInTheDocument()
  })
})
