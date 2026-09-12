import { render, screen, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import LoteoPlanPanel from './LoteoPlanPanel'
import { DXF_LAYERS, type DxfPolygon } from '../types'

const square: DxfPolygon['vertices'] = [
  { x: 0, y: 0 },
  { x: 10, y: 0 },
  { x: 10, y: 10 },
  { x: 0, y: 10 },
]

const polygons: DxfPolygon[] = [
  { id: 'manzana', layer: 'MANZANA', handle: null, vertices: square },
  { id: 'lote', layer: 'LOTES', handle: null, vertices: square },
]

const HINT =
  'Los lotes se dibujan encima de las manzanas. Para seleccionar una manzana, apagá la capa Lotes.'

describe('LoteoPlanPanel', () => {
  it('explains how to select a manzana while the lotes layer is on', () => {
    render(
      <LoteoPlanPanel
        polygons={polygons}
        visibleLayers={new Set(DXF_LAYERS)}
        onVisibleLayersChange={vi.fn()}
        onSelectPolygon={vi.fn()}
      />,
    )

    expect(screen.getByText(HINT)).toBeInTheDocument()
  })

  it('shows a legend with the states present in the plan', () => {
    render(
      <LoteoPlanPanel
        polygons={[
          polygons[0],
          { ...polygons[1], lotState: 'reservado' },
          { id: 'lote-2', layer: 'LOTES', handle: null, vertices: square, lotState: 'vendido' },
        ]}
        visibleLayers={new Set(DXF_LAYERS)}
        onVisibleLayersChange={vi.fn()}
      />,
    )

    const legend = screen.getByRole('list', { name: 'Colores por estado del lote' })
    expect(within(legend).getByText('Reservado')).toBeInTheDocument()
    expect(within(legend).getByText('Vendido')).toBeInTheDocument()
    expect(within(legend).queryByText('Disponible')).not.toBeInTheDocument()
  })

  it('hides the legend while the lotes layer is off', () => {
    render(
      <LoteoPlanPanel
        polygons={[polygons[0], { ...polygons[1], lotState: 'reservado' }]}
        visibleLayers={new Set(['MANZANA'])}
        onVisibleLayersChange={vi.fn()}
      />,
    )

    expect(
      screen.queryByRole('list', { name: 'Colores por estado del lote' }),
    ).not.toBeInTheDocument()
  })

  it('hides the hint once the lotes layer is off', () => {
    render(
      <LoteoPlanPanel
        polygons={polygons}
        visibleLayers={new Set(['MANZANA'])}
        onVisibleLayersChange={vi.fn()}
        onSelectPolygon={vi.fn()}
      />,
    )

    expect(screen.queryByText(HINT)).not.toBeInTheDocument()
  })
})
