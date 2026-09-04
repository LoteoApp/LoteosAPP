import { describe, expect, it } from 'vitest'
import { isValidCuit, normalizeCuit } from './cuit'

describe('normalizeCuit', () => {
  it('drops the separators a CUIT is typed with', () => {
    expect(normalizeCuit('30-71234567-8')).toBe('30712345678')
    expect(normalizeCuit('30 71234567 8')).toBe('30712345678')
    expect(normalizeCuit('30.71234567.8')).toBe('30712345678')
  })

  it('leaves an already normalized CUIT alone', () => {
    expect(normalizeCuit('30712345678')).toBe('30712345678')
  })
})

describe('isValidCuit', () => {
  it('accepts eleven digits', () => {
    expect(isValidCuit('30712345678')).toBe(true)
  })

  it('rejects anything that is not eleven digits', () => {
    expect(isValidCuit('3071234567')).toBe(false)
    expect(isValidCuit('307123456789')).toBe(false)
    expect(isValidCuit('3071234567x')).toBe(false)
    expect(isValidCuit('')).toBe(false)
  })

  it('rejects a CUIT that still carries separators', () => {
    expect(isValidCuit('30-71234567-8')).toBe(false)
  })
})
