import { describe, expect, it } from 'vitest'
import { fitViewBox, panViewBox, viewBoxToString, zoomViewBox } from './fitViewBox'
import type { DxfPolygon } from '../types'

function polygon(vertices: Array<[number, number]>): DxfPolygon {
  return {
    id: 'p1',
    layer: 'LOTES',
    handle: null,
    vertices: vertices.map(([x, y]) => ({ x, y })),
  }
}

describe('fitViewBox', () => {
  it('returns a unit box when there are no polygons', () => {
    expect(fitViewBox([])).toEqual({ x: 0, y: 0, width: 1, height: 1 })
  })

  it('pads the bounding box and flips Y into SVG space', () => {
    const viewBox = fitViewBox([
      polygon([
        [0, 0],
        [10, 0],
        [10, 10],
        [0, 10],
      ]),
    ])

    expect(viewBox.x).toBeCloseTo(-0.8)
    expect(viewBox.y).toBeCloseTo(-10.8)
    expect(viewBox.width).toBeCloseTo(11.6)
    expect(viewBox.height).toBeCloseTo(11.6)
  })

  it('gives a degenerate point a default span', () => {
    const viewBox = fitViewBox([polygon([[5, 5]])])

    expect(viewBox.width).toBeCloseTo(1.16)
    expect(viewBox.height).toBeCloseTo(1.16)
  })

  it('serializes the viewBox attribute', () => {
    expect(viewBoxToString({ x: 1, y: 2, width: 3, height: 4 })).toBe('1 2 3 4')
  })
})

describe('zoomViewBox', () => {
  it('zooms around the center', () => {
    expect(zoomViewBox({ x: 0, y: 0, width: 10, height: 10 }, 0.5)).toEqual({
      x: 2.5,
      y: 2.5,
      width: 5,
      height: 5,
    })
  })
})

describe('panViewBox', () => {
  it('converts pixel deltas into viewBox space', () => {
    expect(panViewBox({ x: 0, y: 0, width: 10, height: 10 }, 50, 0, 100, 100)).toEqual({
      x: -5,
      y: 0,
      width: 10,
      height: 10,
    })
  })

  it('leaves the viewBox alone when the svg has no size', () => {
    const box = { x: 1, y: 2, width: 3, height: 4 }
    expect(panViewBox(box, 10, 10, 0, 100)).toEqual(box)
  })
})
