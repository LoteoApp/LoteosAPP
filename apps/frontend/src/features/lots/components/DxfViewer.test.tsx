import { fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import DxfViewer from './DxfViewer'
import { DXF_LAYERS, type DxfPolygon } from '../types'

const square: DxfPolygon['vertices'] = [
  { x: 0, y: 0 },
  { x: 10, y: 0 },
  { x: 10, y: 10 },
  { x: 0, y: 10 },
]

const polygons: DxfPolygon[] = [
  { id: 'loteo', layer: 'LOTEO', handle: null, vertices: square },
  { id: 'manzana', layer: 'MANZANA', handle: null, vertices: square },
  { id: 'lote', layer: 'LOTES', handle: null, vertices: square },
]

describe('DxfViewer', () => {
  it('shows an empty prompt when there is no geometry', () => {
    render(<DxfViewer polygons={[]} visibleLayers={new Set(DXF_LAYERS)} />)

    expect(
      screen.getByText('El plano aparece acá cuando cargues un DXF.'),
    ).toBeInTheDocument()
    expect(screen.queryByRole('img', { name: 'Plano del loteo' })).not.toBeInTheDocument()
  })

  it('draws the visible layers', () => {
    render(
      <DxfViewer polygons={polygons} visibleLayers={new Set(['LOTEO', 'LOTES'])} />,
    )

    const svg = screen.getByRole('img', { name: 'Plano del loteo' })
    expect(svg).toBeInTheDocument()
    expect(svg.querySelector('[aria-label="Loteo"]')).not.toBeNull()
    expect(svg.querySelector('[aria-label="Lotes"]')).not.toBeNull()
    expect(svg.querySelector('[aria-label="Manzana"]')).toBeNull()
  })

  it('zooms in and fits the drawing again', async () => {
    const user = userEvent.setup()
    render(<DxfViewer polygons={polygons} visibleLayers={new Set(DXF_LAYERS)} />)

    const svg = screen.getByRole('img', { name: 'Plano del loteo' })
    const fitted = svg.getAttribute('viewBox')

    await user.click(screen.getByRole('button', { name: 'Acercar' }))
    expect(svg.getAttribute('viewBox')).not.toBe(fitted)

    await user.click(screen.getByRole('button', { name: 'Alejar' }))
    await user.click(screen.getByRole('button', { name: 'Ajustar al plano' }))
    expect(svg.getAttribute('viewBox')).toBe(fitted)
  })

  it('pans the drawing with the pointer', () => {
    Element.prototype.setPointerCapture = vi.fn()
    Element.prototype.releasePointerCapture = vi.fn()
    Element.prototype.hasPointerCapture = vi.fn().mockReturnValue(true)

    render(<DxfViewer polygons={polygons} visibleLayers={new Set(DXF_LAYERS)} />)

    const svg = screen.getByRole('img', { name: 'Plano del loteo' })
    vi.spyOn(svg, 'getBoundingClientRect').mockReturnValue({
      width: 100,
      height: 100,
      x: 0,
      y: 0,
      top: 0,
      left: 0,
      bottom: 100,
      right: 100,
      toJSON() {
        return {}
      },
    })
    const before = svg.getAttribute('viewBox')

    fireEvent.pointerDown(svg, { clientX: 50, clientY: 50, pointerId: 1 })
    fireEvent.pointerMove(svg, { clientX: 70, clientY: 40, pointerId: 1 })
    fireEvent.pointerUp(svg, { clientX: 70, clientY: 40, pointerId: 1 })

    expect(svg.getAttribute('viewBox')).not.toBe(before)
  })

  it('ignores pointer movement before a drag starts', () => {
    render(<DxfViewer polygons={polygons} visibleLayers={new Set(DXF_LAYERS)} />)

    const svg = screen.getByRole('img', { name: 'Plano del loteo' })
    const before = svg.getAttribute('viewBox')
    fireEvent.pointerMove(svg, { clientX: 80, clientY: 80, pointerId: 1 })
    expect(svg.getAttribute('viewBox')).toBe(before)
  })

  it('does not pan when the svg has no layout size', () => {
    Element.prototype.setPointerCapture = vi.fn()
    Element.prototype.releasePointerCapture = vi.fn()
    Element.prototype.hasPointerCapture = vi.fn().mockReturnValue(false)

    render(<DxfViewer polygons={polygons} visibleLayers={new Set(DXF_LAYERS)} />)

    const svg = screen.getByRole('img', { name: 'Plano del loteo' })
    vi.spyOn(svg, 'getBoundingClientRect').mockReturnValue({
      width: 0,
      height: 0,
      x: 0,
      y: 0,
      top: 0,
      left: 0,
      bottom: 0,
      right: 0,
      toJSON() {
        return {}
      },
    })
    const before = svg.getAttribute('viewBox')

    fireEvent.pointerDown(svg, { clientX: 10, clientY: 10, pointerId: 1 })
    fireEvent.pointerMove(svg, { clientX: 40, clientY: 40, pointerId: 1 })
    fireEvent.pointerCancel(svg, { pointerId: 1 })

    expect(svg.getAttribute('viewBox')).toBe(before)
  })

  it('refits the drawing when the visible layers change', async () => {
    const user = userEvent.setup()
    const { rerender } = render(
      <DxfViewer polygons={polygons} visibleLayers={new Set(['LOTEO'])} />,
    )

    await user.click(screen.getByRole('button', { name: 'Acercar' }))
    const zoomed = screen.getByRole('img', { name: 'Plano del loteo' }).getAttribute('viewBox')

    rerender(<DxfViewer polygons={polygons} visibleLayers={new Set(['LOTES'])} />)
    expect(
      screen.getByRole('img', { name: 'Plano del loteo' }).getAttribute('viewBox'),
    ).not.toBe(zoomed)
  })
})
