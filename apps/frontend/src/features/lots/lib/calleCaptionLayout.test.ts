import { describe, expect, it } from 'vitest'
import { calleCaptionLayout } from './calleCaptionLayout'

const horizontal = [
  { x: 0, y: 0 },
  { x: 100, y: 0 },
  { x: 100, y: 4 },
  { x: 0, y: 4 },
]

const vertical = [
  { x: 0, y: 0 },
  { x: 4, y: 0 },
  { x: 4, y: 100 },
  { x: 0, y: 100 },
]

describe('calleCaptionLayout', () => {
  it('places a horizontal street name along the strip in SVG space', () => {
    const layout = calleCaptionLayout(horizontal, 'San Martín')

    expect(layout).not.toBeNull()
    expect(layout?.x).toBeCloseTo(50)
    expect(layout?.y).toBeCloseTo(-2)
    expect(layout?.rotate).toBeCloseTo(0)
    expect(layout?.fontSize).toBeGreaterThan(0)
    expect(layout?.fontSize).toBeLessThan(4)
  })

  it('rotates a north-south street so the name follows the strip', () => {
    const layout = calleCaptionLayout(vertical, 'Belgrano')

    expect(layout).not.toBeNull()
    expect(layout?.x).toBeCloseTo(2)
    expect(layout?.y).toBeCloseTo(-50)
    expect(layout?.rotate).toBeCloseTo(-90)
  })

  it('keeps text upright when the long edge points left', () => {
    const rightToLeft = [
      { x: 100, y: 0 },
      { x: 0, y: 0 },
      { x: 0, y: 4 },
      { x: 100, y: 4 },
    ]

    const layout = calleCaptionLayout(rightToLeft, 'Mitre')
    expect(layout?.rotate).toBeCloseTo(0)
  })

  it('follows a diagonal street after the SVG Y flip', () => {
    const sqrt2 = Math.SQRT2
    const u = { x: 1 / sqrt2, y: 1 / sqrt2 }
    const n = { x: -1 / sqrt2, y: 1 / sqrt2 }
    const halfLength = 20
    const halfWidth = 1
    const vertices = [
      { x: halfLength * u.x + halfWidth * n.x, y: halfLength * u.y + halfWidth * n.y },
      { x: halfLength * u.x - halfWidth * n.x, y: halfLength * u.y - halfWidth * n.y },
      { x: -halfLength * u.x - halfWidth * n.x, y: -halfLength * u.y - halfWidth * n.y },
      { x: -halfLength * u.x + halfWidth * n.x, y: -halfLength * u.y + halfWidth * n.y },
    ]

    const layout = calleCaptionLayout(vertices, 'Álamos')
    expect(layout?.x).toBeCloseTo(0)
    expect(layout?.y).toBeCloseTo(0)
    expect(layout?.rotate).toBeCloseTo(-45)
    expect(layout?.fontSize).toBeLessThan(2)
  })

  it('shrinks the font so a longer name still fits the length of the strip', () => {
    const shortStreet = [
      { x: 0, y: 0 },
      { x: 20, y: 0 },
      { x: 20, y: 4 },
      { x: 0, y: 4 },
    ]
    const short = calleCaptionLayout(shortStreet, 'A')
    const long = calleCaptionLayout(shortStreet, 'Avenida San Martín')

    expect(short?.fontSize).toBeGreaterThan(long?.fontSize ?? Infinity)
  })

  it('returns null when the ring has no area or the name is empty', () => {
    expect(calleCaptionLayout([], 'Mitre')).toBeNull()
    expect(calleCaptionLayout(horizontal, '')).toBeNull()
    expect(
      calleCaptionLayout(
        [
          { x: 1, y: 1 },
          { x: 1, y: 1 },
          { x: 1, y: 1 },
        ],
        'Mitre',
      ),
    ).toBeNull()
  })
})
