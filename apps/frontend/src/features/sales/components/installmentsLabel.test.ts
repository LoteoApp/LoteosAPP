import { describe, expect, it } from 'vitest'
import { installmentsLabel } from './installmentsLabel'

// Intl formats with non-breaking spaces; compare on plain ones.
const plain = (value: string) => value.replace(/\s/g, ' ')

describe('installmentsLabel', () => {
  it('prints one term when every cuota is the same', () => {
    expect(plain(installmentsLabel(12, 13750, 13750, 'USD'))).toBe('12 × US$ 13.750,00')
  })

  it('prints the last cuota apart when it absorbs the remainder', () => {
    expect(plain(installmentsLabel(3, 33.33, 33.34, 'USD'))).toBe('2 × US$ 33,33 + 1 × US$ 33,34')
  })

  it('never prints "0 ×" for a single cuota', () => {
    expect(plain(installmentsLabel(1, 1124.99, 1124.99, 'ARS'))).toBe('1 × $ 1.124,99')
  })
})
