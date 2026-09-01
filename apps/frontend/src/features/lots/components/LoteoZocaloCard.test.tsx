import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { describe, expect, it } from 'vitest'
import LoteoZocaloCard from './LoteoZocaloCard'
import type { LoteoSummary } from '../types'

function summary(overrides: Partial<LoteoSummary> = {}): LoteoSummary {
  return {
    id: 'loteo-1',
    nombre: 'Loteo Las Acacias',
    ubicacion: 'Río Ceballos, Córdoba',
    descripcion: 'Loteo residencial sobre ruta E-53.',
    cantidadManzanas: 12,
    cantidadLotes: 148,
    cantidadCalles: 8,
    tienePlano: true,
    tieneDxf: true,
    fechaCreacion: '2026-01-10T12:00:00Z',
    ...overrides,
  }
}

function renderCard(loteo: LoteoSummary) {
  render(
    <MemoryRouter>
      <LoteoZocaloCard loteo={loteo} />
    </MemoryRouter>,
  )
}

describe('LoteoZocaloCard', () => {
  it('shows the name, location, description and the three metrics', () => {
    renderCard(summary())

    expect(screen.getByText('Loteo Las Acacias')).toBeInTheDocument()
    expect(screen.getByText('Río Ceballos, Córdoba')).toBeInTheDocument()
    expect(screen.getByText('Loteo residencial sobre ruta E-53.')).toBeInTheDocument()
    expect(screen.getByText('148')).toBeInTheDocument()
    expect(screen.getByText('Lotes')).toBeInTheDocument()
    expect(screen.getByText('12')).toBeInTheDocument()
    expect(screen.getByText('Manzanas')).toBeInTheDocument()
    expect(screen.getByText('8')).toBeInTheDocument()
    expect(screen.getByText('Calles')).toBeInTheDocument()
  })

  it('links the whole band to the loteo detail', () => {
    renderCard(summary({ id: 'loteo-42' }))

    expect(
      screen.getByRole('link', { name: 'Ver detalle de Loteo Las Acacias' }),
    ).toHaveAttribute('href', '/lotes/loteo-42')
  })

  it('marks the plan status', () => {
    renderCard(summary({ tienePlano: false }))

    expect(screen.getByText('Plano pendiente')).toBeInTheDocument()
    expect(screen.getByText('Sin plano')).toBeInTheDocument()
  })
})
