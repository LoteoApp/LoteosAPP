import { describe, expect, it } from 'vitest'
import { normalizeText } from './normalizeText'

describe('normalizeText', () => {
  it('lowercases the value', () => {
    expect(normalizeText('GOMEZ')).toBe('gomez')
  })

  it('strips accents so accented and plain text compare equal', () => {
    expect(normalizeText('Pérez')).toBe('perez')
    expect(normalizeText('José María Ñandú')).toBe('jose maria nandu')
  })

  it('lets accented and unaccented input match each other', () => {
    expect(normalizeText('Gómez')).toBe(normalizeText('gomez'))
  })
})
