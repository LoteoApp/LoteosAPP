import { render, screen } from '@testing-library/react'
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
