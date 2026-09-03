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

  it('draws the lote and manzana numbers at the center of each polygon', () => {
    const labeled = polygons.map((polygon) => {
      if (polygon.layer === 'LOTES') return { ...polygon, caption: '7' }
      if (polygon.layer === 'MANZANA') return { ...polygon, caption: 'A' }
      return polygon
    })

    render(<DxfViewer polygons={labeled} visibleLayers={new Set(['MANZANA', 'LOTES'])} />)

    const svg = screen.getByRole('img', { name: 'Plano del loteo' })
    const captions = [...svg.querySelectorAll('text')].map((node) => node.textContent)
    expect(captions).toEqual(['A', '7'])
  })

  it('draws a calle name along the street without a number badge', () => {
    const street: DxfPolygon = {
      id: 'calle-1',
      layer: 'CALLE',
      handle: null,
      vertices: [
        { x: 0, y: 0 },
        { x: 100, y: 0 },
        { x: 100, y: 4 },
        { x: 0, y: 4 },
      ],
      caption: 'Los Álamos',
    }

    render(<DxfViewer polygons={[street]} visibleLayers={new Set(['CALLE'])} />)

    const svg = screen.getByRole('img', { name: 'Plano del loteo' })
    const caption = svg.querySelector('text')
    expect(caption?.textContent).toBe('Los Álamos')
    expect(caption?.getAttribute('transform')).toMatch(/rotate\(0 /)
    expect(svg.querySelector('circle')).toBeNull()
  })

  it('hides the lote number when the lotes layer is off', () => {
    const labeled = polygons.map((polygon) =>
      polygon.layer === 'LOTES' ? { ...polygon, caption: '7' } : polygon,
    )

    render(<DxfViewer polygons={labeled} visibleLayers={new Set(['LOTEO'])} />)

    const svg = screen.getByRole('img', { name: 'Plano del loteo' })
    expect(svg.querySelector('text')).toBeNull()
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

  it('keeps the current view when the visible layers change', async () => {
    const user = userEvent.setup()
    const { rerender } = render(
      <DxfViewer polygons={polygons} visibleLayers={new Set(['LOTEO'])} />,
    )

    await user.click(screen.getByRole('button', { name: 'Acercar' }))
    const zoomed = screen.getByRole('img', { name: 'Plano del loteo' }).getAttribute('viewBox')

    rerender(<DxfViewer polygons={polygons} visibleLayers={new Set(['LOTES'])} />)
    expect(
      screen.getByRole('img', { name: 'Plano del loteo' }).getAttribute('viewBox'),
    ).toBe(zoomed)
  })

  it('refits the drawing when a new set of polygons arrives', async () => {
    const user = userEvent.setup()
    const { rerender } = render(
      <DxfViewer polygons={polygons} visibleLayers={new Set(DXF_LAYERS)} />,
    )

    const svg = screen.getByRole('img', { name: 'Plano del loteo' })
    const fitted = svg.getAttribute('viewBox')
    await user.click(screen.getByRole('button', { name: 'Acercar' }))
    expect(svg.getAttribute('viewBox')).not.toBe(fitted)

    rerender(<DxfViewer polygons={[...polygons]} visibleLayers={new Set(DXF_LAYERS)} />)
    expect(
      screen.getByRole('img', { name: 'Plano del loteo' }).getAttribute('viewBox'),
    ).toBe(fitted)
  })

  it('zooms with the wheel', () => {
    render(<DxfViewer polygons={polygons} visibleLayers={new Set(DXF_LAYERS)} />)

    const svg = screen.getByRole('img', { name: 'Plano del loteo' })
    const before = svg.getAttribute('viewBox')

    fireEvent.wheel(svg, { deltaY: 120 })

    expect(svg.getAttribute('viewBox')).not.toBe(before)
  })

  it('zooms with the wheel when the geometry arrives after mounting', () => {
    const { rerender } = render(
      <DxfViewer polygons={[]} visibleLayers={new Set(DXF_LAYERS)} />,
    )

    rerender(<DxfViewer polygons={polygons} visibleLayers={new Set(DXF_LAYERS)} />)

    const svg = screen.getByRole('img', { name: 'Plano del loteo' })
    const before = svg.getAttribute('viewBox')

    fireEvent.wheel(svg, { deltaY: 120 })

    expect(svg.getAttribute('viewBox')).not.toBe(before)
  })

  it('zooms when two pointers spread apart', () => {
    Element.prototype.setPointerCapture = vi.fn()
    Element.prototype.releasePointerCapture = vi.fn()
    Element.prototype.hasPointerCapture = vi.fn().mockReturnValue(true)

    render(<DxfViewer polygons={polygons} visibleLayers={new Set(DXF_LAYERS)} />)

    const svg = screen.getByRole('img', { name: 'Plano del loteo' })
    const before = svg.getAttribute('viewBox')

    fireEvent.pointerDown(svg, { clientX: 0, clientY: 0, pointerId: 1 })
    fireEvent.pointerDown(svg, { clientX: 10, clientY: 0, pointerId: 2 })
    fireEvent.pointerMove(svg, { clientX: 20, clientY: 0, pointerId: 2 })
    expect(svg.getAttribute('viewBox')).toBe(before)

    fireEvent.pointerMove(svg, { clientX: 40, clientY: 0, pointerId: 2 })
    const pinched = svg.getAttribute('viewBox')
    expect(pinched).not.toBe(before)

    const [, , width] = String(pinched).split(' ').map(Number)
    const [, , widthBefore] = String(before).split(' ').map(Number)
    expect(width).toBeLessThan(widthBefore)
  })

  it('keeps the drawing as an image in read-only mode', () => {
    render(<DxfViewer polygons={polygons} visibleLayers={new Set(DXF_LAYERS)} />)

    expect(screen.getByRole('img', { name: 'Plano del loteo' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Loteo' })).not.toBeInTheDocument()
  })

  it('exposes named buttons for each polygon when selection is enabled', async () => {
    const user = userEvent.setup()
    const onSelectPolygon = vi.fn()
    const polygonLabels = new Map([
      ['loteo', 'Contorno del loteo'],
      ['manzana', 'Manzana 1'],
      ['lote', 'Lote 7'],
    ])

    render(
      <DxfViewer
        polygons={polygons}
        visibleLayers={new Set(DXF_LAYERS)}
        selectedPolygonId="lote"
        onSelectPolygon={onSelectPolygon}
        polygonLabels={polygonLabels}
      />,
    )

    expect(screen.getByRole('group', { name: 'Plano del loteo' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Lote 7' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'Manzana 1' })).toHaveAttribute(
      'aria-pressed',
      'false',
    )

    await user.click(screen.getByRole('button', { name: 'Manzana 1' }))
    expect(onSelectPolygon).toHaveBeenCalledWith('manzana')
  })

  it('selects a polygon with Enter', async () => {
    const user = userEvent.setup()
    const onSelectPolygon = vi.fn()

    render(
      <DxfViewer
        polygons={polygons}
        visibleLayers={new Set(DXF_LAYERS)}
        onSelectPolygon={onSelectPolygon}
        polygonLabels={new Map([['lote', 'Lote 7']])}
      />,
    )

    screen.getByRole('button', { name: 'Lote 7' }).focus()
    await user.keyboard('{Enter}')
    expect(onSelectPolygon).toHaveBeenCalledWith('lote')

    onSelectPolygon.mockClear()
    await user.keyboard(' ')
    expect(onSelectPolygon).toHaveBeenCalledWith('lote')

    onSelectPolygon.mockClear()
    await user.keyboard('a')
    expect(onSelectPolygon).not.toHaveBeenCalled()
  })

  it('clears the selection when the background of the drawing is clicked', async () => {
    const user = userEvent.setup()
    const onSelectPolygon = vi.fn()

    render(
      <DxfViewer
        polygons={polygons}
        visibleLayers={new Set(DXF_LAYERS)}
        selectedPolygonId="lote"
        onSelectPolygon={onSelectPolygon}
      />,
    )

    await user.click(screen.getByRole('group', { name: 'Plano del loteo' }))
    expect(onSelectPolygon).toHaveBeenCalledWith(null)
  })

  it('does not select after a drag', () => {
    const onSelectPolygon = vi.fn()

    render(
      <DxfViewer
        polygons={polygons}
        visibleLayers={new Set(DXF_LAYERS)}
        onSelectPolygon={onSelectPolygon}
        polygonLabels={new Map([['lote', 'Lote 7']])}
      />,
    )

    const svg = screen.getByRole('group', { name: 'Plano del loteo' })
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

    fireEvent.pointerDown(svg, { clientX: 50, clientY: 50, pointerId: 1 })
    fireEvent.pointerMove(svg, { clientX: 70, clientY: 40, pointerId: 1 })
    fireEvent.click(screen.getByRole('button', { name: 'Lote 7' }))

    expect(onSelectPolygon).not.toHaveBeenCalled()
  })
})
