import { describe, expect, it } from 'vitest'
import { listInmobiliarias, MOCK_INMOBILIARIAS } from './list-inmobiliarias'

describe('listInmobiliarias', () => {
  it('returns every inmobiliaria with an id and a name', () => {
    const inmobiliarias = listInmobiliarias()

    expect(inmobiliarias).toEqual(MOCK_INMOBILIARIAS)
    expect(inmobiliarias.length).toBeGreaterThan(1)
    expect(new Set(inmobiliarias.map((item) => item.id)).size).toBe(inmobiliarias.length)
    expect(inmobiliarias.every((item) => item.razonSocial.length > 0)).toBe(true)
  })
})
