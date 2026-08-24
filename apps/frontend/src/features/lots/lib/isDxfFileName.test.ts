import { describe, expect, it } from 'vitest'
import { isDxfFileName } from './isDxfFileName'

describe('isDxfFileName', () => {
  it('accepts .dxf regardless of case', () => {
    expect(isDxfFileName('plano.dxf')).toBe(true)
    expect(isDxfFileName('Plano.DXF')).toBe(true)
  })

  it('rejects other extensions', () => {
    expect(isDxfFileName('plano.txt')).toBe(false)
    expect(isDxfFileName('plano.dxf.bak')).toBe(false)
    expect(isDxfFileName('dxf')).toBe(false)
  })
})
