import { describe, expect, it } from 'vitest'
import { loteCaptionLayout } from './loteCaptionLayout'

const square = [
  { x: 0, y: 0 },
  { x: 10, y: 0 },
  { x: 10, y: 10 },
  { x: 0, y: 10 },
]

describe('loteCaptionLayout', () => {
  it('places the caption at the center of the polygon in SVG space', () => {
    expect(loteCaptionLayout(square)).toEqual({ x: 5, y: -5, fontSize: 2.2 })
  })

  it('returns null when the ring has no area', () => {
    expect(loteCaptionLayout([])).toBeNull()
    expect(
      loteCaptionLayout([
        { x: 1, y: 1 },
        { x: 1, y: 1 },
        { x: 1, y: 1 },
      ]),
    ).toBeNull()
  })
})
