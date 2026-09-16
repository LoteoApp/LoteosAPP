import { describe, expect, it } from 'vitest'
import { toSvgPoint } from './toSvgPoint'

describe('toSvgPoint', () => {
  it('flips Y and leaves X alone', () => {
    expect(toSvgPoint({ x: 3, y: 7 })).toEqual({ x: 3, y: -7 })
  })

  it('flips negative Y back up', () => {
    expect(toSvgPoint({ x: -2, y: -5 })).toEqual({ x: -2, y: 5 })
  })
})
