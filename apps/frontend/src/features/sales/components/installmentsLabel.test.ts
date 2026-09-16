import { describe, expect, it } from 'vitest'
import { installmentsLabel } from './installmentsLabel'

// Intl formats with non-breaking spaces; compare on plain ones.
const plain = (value: string) => value.replace(/\s/g, ' ')

describe('installmentsLabel', () => {
  it('prints the count and the regular cuota when every cuota is the same', () => {
    expect(plain(installmentsLabel(12, 13750, 13750, 'USD'))).toBe('12 de US$ 13.750,00')
  })

  it('notes the last cuota when it absorbs the remainder', () => {
    expect(plain(installmentsLabel(12, 383.33, 383.37, 'USD'))).toBe(
      '12 de US$ 383,33 (la última de US$ 383,37)',
    )
  })

  it('never notes a last cuota for a single one', () => {
    expect(plain(installmentsLabel(1, 1124.99, 1124.99, 'ARS'))).toBe('1 de $ 1.124,99')
  })
})
