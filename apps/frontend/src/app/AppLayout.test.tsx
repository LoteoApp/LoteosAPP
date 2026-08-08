import { render, screen } from '@testing-library/react'
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
})
