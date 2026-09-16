import { describe, expect, it } from 'vitest'
import { polygonToSvgPath } from './polygonToSvgPath'

describe('polygonToSvgPath', () => {
  it('returns an empty string when there are no vertices', () => {
    expect(polygonToSvgPath([])).toBe('')
  })

  it('flips Y so DXF up becomes SVG down and closes the ring', () => {
    expect(
      polygonToSvgPath([
        { x: 0, y: 0 },
        { x: 10, y: 0 },
        { x: 10, y: 10 },
        { x: 0, y: 10 },
      ]),
    ).toBe('M0 0L10 0L10 -10L0 -10Z')
  })
})
