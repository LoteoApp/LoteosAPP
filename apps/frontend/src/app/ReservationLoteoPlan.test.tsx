import { render, screen, within } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import ReservationLoteoPlan from './ReservationLoteoPlan'
import type { LoteoDetail } from '../features/lots/types'

const triangle = [{ x: 0, y: 0 }, { x: 10, y: 0 }, { x: 10, y: 10 }]
const loteo: LoteoDetail = {
  id: 'loteo-1', nombre: 'Las Acacias', ubicacion: 'Córdoba', descripcion: '', contorno: triangle,
  manzanas: [],
  lotes: [
    { id: 'lot-1', manzanaId: 'block-1', numero: '7', estado: 'disponible', precio: 120000, moneda: 'USD', superficie: 300, caracteristicas: '', poligono: triangle },
    { id: 'lot-2', manzanaId: 'block-1', numero: '8', estado: 'disponible', precio: 130000, moneda: 'USD', superficie: 310, caracteristicas: '', poligono: triangle },
  ],
  calles: [], fechaCreacion: '2026-08-20T12:00:00Z',
}

describe('ReservationLoteoPlan', () => {
  it('keeps every lot state and caption in full mode', () => {
    render(<ReservationLoteoPlan loteo={loteo} selectedLoteId="lot-1" />)
    const plan = within(screen.getByRole('img', { name: 'Plano del loteo' }))
    const selected = plan.getByLabelText('Lote 7')
    const other = plan.getByLabelText('Lote 8')

    expect(selected).toHaveAttribute('stroke-width', '2.5')
    expect(selected).toHaveAttribute('aria-description', 'Disponible')
    expect(other).toHaveAttribute('stroke-width', '1')
    expect(other).toHaveAttribute('aria-description', 'Disponible')
    expect(plan.getByText('7')).toBeInTheDocument()
    expect(plan.getByText('8')).toBeInTheDocument()
  })

  it('shows only the selected lot state and caption in reference mode while preserving context', () => {
    render(<ReservationLoteoPlan loteo={loteo} selectedLoteId="lot-1" variant="reference" />)
    const plan = within(screen.getByRole('img', { name: 'Plano del loteo' }))
    const selected = plan.getByLabelText('Lote 7')
    const other = plan.getByLabelText('Lote 8')
    const contour = plan.getByLabelText('Contorno del loteo')

    expect(selected).toHaveAttribute('stroke-width', '2.5')
    expect(selected).toHaveAttribute('aria-description', 'Disponible')
    expect(other).toHaveAttribute('fill', 'var(--plan-block)')
    expect(other).not.toHaveAttribute('aria-description')
    expect(plan.getByText('7')).toBeInTheDocument()
    expect(plan.queryByText('8')).not.toBeInTheDocument()
    expect(contour).toHaveAttribute('fill', 'none')
  })
})
