import { describe, expect, it } from 'vitest'
import { installmentsLabel } from './installmentsLabel'

// Intl formats with non-breaking spaces; compare on plain ones.
const plain = (value: string) => value.replace(/\s/g, ' ')

describe('installmentsLabel', () => {
  it('prints the count and the cuota', () => {
    expect(plain(installmentsLabel(12, 383.33, 'USD'))).toBe('12 de US$ 383,33')
    expect(plain(installmentsLabel(1, 1124.99, 'ARS'))).toBe('1 de $ 1.124,99')
  })
})
