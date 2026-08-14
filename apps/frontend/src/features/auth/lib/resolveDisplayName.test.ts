import { describe, expect, it } from 'vitest'
import { resolveDisplayName } from './resolveDisplayName'

describe('resolveDisplayName', () => {
  it('prefers the preferred username when present', () => {
    expect(
      resolveDisplayName({
        preferred_username: 'lzorzoli',
        name: 'Leonel Zorzoli',
        email: 'leonel@loteosapp.com',
      }),
    ).toBe('lzorzoli')
  })

  it('falls back to the name when the preferred username is an empty string', () => {
    expect(
      resolveDisplayName({
        preferred_username: '',
        name: 'Leonel Zorzoli',
        email: 'leonel@loteosapp.com',
      }),
    ).toBe('Leonel Zorzoli')
  })

  it('falls back to the email when neither username nor name are set', () => {
    expect(
      resolveDisplayName({
        preferred_username: '',
        name: '',
        email: 'leonel@loteosapp.com',
      }),
    ).toBe('leonel@loteosapp.com')
  })

  it('uses the provided fallback when the profile is missing entirely', () => {
    expect(resolveDisplayName(undefined, 'Usuario')).toBe('Usuario')
  })

  it('returns an empty string by default when nothing is available', () => {
    expect(resolveDisplayName({ preferred_username: '', name: '', email: '' })).toBe('')
  })
})
