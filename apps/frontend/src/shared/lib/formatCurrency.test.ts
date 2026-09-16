import { describe, expect, it } from 'vitest'
import { formatCurrency } from './formatCurrency'

describe('formatCurrency', () => {
  it('formats an amount with its ISO currency in es-AR', () => {
    const formatted = formatCurrency(150000, 'USD')

    expect(formatted).toMatch(/150\.000/)
    expect(formatted).toMatch(/US\$|USD/)
  })

  it('falls back to "<code> <number>" when the currency code is not valid', () => {
    const formatted = formatCurrency(150000, 'PESOS')

    expect(formatted).toBe('PESOS 150.000,00')
  })

  it('formats the bare number when no currency is set', () => {
    expect(formatCurrency(150000, '')).toBe('150.000,00')
    expect(formatCurrency(1234.5, '   ')).toBe('1.234,50')
  })
})
