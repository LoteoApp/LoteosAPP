import { describe, expect, it } from 'vitest'
import { formatPercent } from './formatPercent'

describe('formatPercent', () => {
  it('formats a rate in es-AR with the % suffix', () => {
    expect(formatPercent(10)).toBe('10 %')
    expect(formatPercent(0)).toBe('0 %')
  })

  it('keeps up to four decimals', () => {
    expect(formatPercent(12.5)).toBe('12,5 %')
    expect(formatPercent(1.23456)).toBe('1,2346 %')
  })
})
