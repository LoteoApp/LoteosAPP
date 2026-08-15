import { describe, expect, it } from 'vitest'
import type { DxfPoint } from '../types'
import { isSimplePolygon, polygonsOverlap } from './polygonGeometry'

function points(coords: [number, number][]): DxfPoint[] {
  return coords.map(([x, y]) => ({ x, y }))
}

const square = points([
  [0, 0],
  [10, 0],
  [10, 10],
  [0, 10],
])

describe('isSimplePolygon', () => {
  it('returns true for a polygon whose edges do not cross', () => {
    expect(isSimplePolygon(square)).toBe(true)
  })

  it('returns false for a bowtie polygon whose edges cross', () => {
    const bowtie = points([
      [0, 0],
      [10, 10],
      [10, 0],
      [0, 10],
    ])

    expect(isSimplePolygon(bowtie)).toBe(false)
  })

  it('returns false for a ring with fewer than 3 vertices', () => {
    expect(isSimplePolygon(points([[0, 0], [10, 0]]))).toBe(false)
  })

  it('returns false for a bowtie at Gauss-Krüger-scale coordinates', () => {
    // A fixed absolute epsilon would misjudge orientation here: the ULP at
    // a magnitude of ~5.45 million is already close to 1e-9.
    const bowtie = points([
      [5454366.14, 6486276.57],
      [5454376.14, 6486286.57],
      [5454376.14, 6486276.57],
      [5454366.14, 6486286.57],
    ])

    expect(isSimplePolygon(bowtie)).toBe(false)
  })
})

describe('polygonsOverlap', () => {
  it('returns false for two polygons that do not touch', () => {
    const other = points([
      [100, 100],
      [110, 100],
      [110, 110],
      [100, 110],
    ])

    expect(polygonsOverlap(square, other)).toBe(false)
  })

  it('returns true when polygon edges cross', () => {
    const other = points([
      [5, 5],
      [15, 5],
      [15, 15],
      [5, 15],
    ])

    expect(polygonsOverlap(square, other)).toBe(true)
  })

  it('returns true when one polygon fully contains the other with no edge crossing', () => {
    const inner = points([
      [2, 2],
      [8, 2],
      [8, 8],
      [2, 8],
    ])

    expect(polygonsOverlap(square, inner)).toBe(true)
    expect(polygonsOverlap(inner, square)).toBe(true)
  })

  it('returns false for adjacent polygons that only share an edge', () => {
    const neighbor = points([
      [10, 0],
      [20, 0],
      [20, 10],
      [10, 10],
    ])

    expect(polygonsOverlap(square, neighbor)).toBe(false)
  })

  it('returns false for adjacent polygons that only share a corner vertex', () => {
    const neighbor = points([
      [10, 10],
      [20, 10],
      [20, 20],
      [10, 20],
    ])

    expect(polygonsOverlap(square, neighbor)).toBe(false)
  })

  it('returns false for adjacent lots at real Gauss-Krüger coordinates whose shared corner was independently redrawn', () => {
    // Real subdivision plan (issue #49 manual check): three lots meet at one
    // point, but each polyline was traced separately, so the "same" corner
    // differs by ~1e-9 between polygons. A fixed epsilon on the orientation
    // cross-product alone doesn't catch this, because the cross-product
    // terms shrink together with that gap and never cross the threshold —
    // segmentsCrossProperly must check point coincidence directly.
    const a = points([
      [5454393.32834033, 6486200.674055615],
      [5454413.022188838, 6486197.188036259],
      [5454404.307140451, 6486147.953414989],
      [5454384.613291944, 6486151.439434344],
    ])
    const b = points([
      [5454413.022188837, 6486197.188036259],
      [5454432.716037342, 6486193.702016905],
      [5454424.000988954, 6486144.467395635],
      [5454404.307140451, 6486147.953414988],
    ])

    expect(polygonsOverlap(a, b)).toBe(false)
  })

  it('still detects a genuine crossing at Gauss-Krüger-scale coordinates', () => {
    const a = points([
      [5454366.14, 6486276.57],
      [5454376.14, 6486276.57],
      [5454376.14, 6486286.57],
      [5454366.14, 6486286.57],
    ])
    const b = points([
      [5454371.14, 6486281.57],
      [5454381.14, 6486281.57],
      [5454381.14, 6486291.57],
      [5454371.14, 6486291.57],
    ])

    expect(polygonsOverlap(a, b)).toBe(true)
  })
})
