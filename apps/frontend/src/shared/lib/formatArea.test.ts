import { describe, expect, it } from 'vitest'
import { formatArea } from './formatArea'

describe('formatArea', () => {
  it('formats square meters in es-AR with the m² suffix', () => {
    expect(formatArea(300)).toBe('300 m²')
  })

  it('groups thousands and keeps at most two decimals', () => {
    expect(formatArea(1234.5)).toBe('1.234,5 m²')
    expect(formatArea(1234.567)).toBe('1.234,57 m²')
  })
})
