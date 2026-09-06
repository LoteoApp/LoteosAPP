import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { describe, expect, it } from 'vitest'
import LoteoDetailHeader from './LoteoDetailHeader'
import type { LoteoDetail } from '../types'

function detail(overrides: Partial<LoteoDetail> = {}): LoteoDetail {
  return {
    id: 'loteo-1',
    nombre: 'Las Acacias',
    ubicacion: 'Río Ceballos, Córdoba',
    descripcion: 'Sobre ruta E-53.',
    contorno: [],
    manzanas: [
      { id: 'mz-1', numero: '1', tieneAgua: false, tieneCloaca: false, tieneLuz: false, tieneGas: false, calleIds: [], poligono: [] },
      { id: 'mz-2', numero: '2', tieneAgua: false, tieneCloaca: false, tieneLuz: false, tieneGas: false, calleIds: [], poligono: [] },
    ],
    lotes: [],
    calles: [{ id: 'ca-1', nombre: 'Los Álamos', tipo: 'asfalto', poligono: [] }],
    fechaCreacion: '2026-08-20T12:00:00Z',
    ...overrides,
  }
}

function renderHeader(loteo: LoteoDetail, hasPlan: boolean) {
  render(
    <MemoryRouter>
      <LoteoDetailHeader loteo={loteo} hasPlan={hasPlan} />
    </MemoryRouter>,
  )
}

describe('LoteoDetailHeader', () => {
  it('shows the loteo name, location, description and entity counts', () => {
    renderHeader(detail(), true)

    expect(screen.getByRole('heading', { name: 'Las Acacias' })).toBeInTheDocument()
    expect(screen.getByText('Río Ceballos, Córdoba')).toBeInTheDocument()
    expect(screen.getByText('Sobre ruta E-53.')).toBeInTheDocument()
    // 2 manzanas, 0 lotes, 1 calle
    expect(screen.getByText('Manzanas').previousSibling).toHaveTextContent('2')
    expect(screen.getByText('Calles').previousSibling).toHaveTextContent('1')
  })

  it('reflects the plan status through the badge', () => {
    renderHeader(detail(), false)
    expect(screen.getByText('Plano pendiente')).toBeInTheDocument()
  })

  it('links back to the listing', () => {
    renderHeader(detail(), true)

    expect(screen.getByRole('link', { name: 'Volver al listado' })).toHaveAttribute(
      'href',
      '/lotes',
    )
  })
})
