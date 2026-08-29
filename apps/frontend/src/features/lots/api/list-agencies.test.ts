import { describe, expect, it } from 'vitest'
import { listAgencies, MOCK_AGENCIES } from './list-agencies'

describe('listAgencies', () => {
  it('returns every agency with an id and a name', () => {
    const agencies = listAgencies()

    expect(agencies).toEqual(MOCK_AGENCIES)
    expect(agencies.length).toBeGreaterThan(1)
    expect(new Set(agencies.map((item) => item.id)).size).toBe(agencies.length)
    expect(agencies.every((item) => item.businessName.length > 0)).toBe(true)
  })
})
