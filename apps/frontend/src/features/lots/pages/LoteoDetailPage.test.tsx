import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router'
import { describe, expect, it } from 'vitest'
import LoteoDetailPage from './LoteoDetailPage'

function renderDetail(path: string) {
  render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/lotes/:loteoId" element={<LoteoDetailPage />} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('LoteoDetailPage', () => {
  it('shows the loteo id and a placeholder message', () => {
    renderDetail('/lotes/loteo-7')

    expect(screen.getByRole('heading', { name: 'Loteo loteo-7' })).toBeInTheDocument()
    expect(
      screen.getByText('El detalle del loteo estará disponible próximamente.'),
    ).toBeInTheDocument()
  })

  it('links back to the list', () => {
    renderDetail('/lotes/loteo-7')

    expect(screen.getByRole('link', { name: 'Volver al listado' })).toHaveAttribute(
      'href',
      '/lotes',
    )
  })
})
