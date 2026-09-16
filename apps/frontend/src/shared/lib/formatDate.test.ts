import { describe, expect, it } from 'vitest'
import { formatDate } from './formatDate'

describe('formatDate', () => {
  it('renders an RFC 3339 timestamp as a short es-AR date', () => {
    expect(formatDate('2026-08-20T12:00:00Z')).toBe('20/08/2026')
  })

  it('returns an empty string for an unparseable value', () => {
    expect(formatDate('not a date')).toBe('')
    expect(formatDate('')).toBe('')
  })
})
