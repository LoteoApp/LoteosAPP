import { describe, expect, it } from 'vitest'
import { resolveDisplayName } from './resolveDisplayName'

describe('resolveDisplayName', () => {
  it('prefers the full name stored in the user metadata', () => {
    expect(
      resolveDisplayName({
        email: 'leonel@loteosapp.com',
        user_metadata: { full_name: 'Leonel Zorzoli' },
      }),
    ).toBe('Leonel Zorzoli')
  })

  it('falls back to the email when the full name is an empty string', () => {
    expect(
      resolveDisplayName({
        email: 'leonel@loteosapp.com',
        user_metadata: { full_name: '' },
      }),
    ).toBe('leonel@loteosapp.com')
  })

  it('falls back to the email when the user has no metadata', () => {
    expect(resolveDisplayName({ email: 'leonel@loteosapp.com' })).toBe(
      'leonel@loteosapp.com',
    )
  })

  it('uses the provided fallback when there is no user', () => {
    expect(resolveDisplayName(null, 'Usuario')).toBe('Usuario')
  })

  it('returns an empty string by default when nothing is available', () => {
    expect(resolveDisplayName({ email: '', user_metadata: { full_name: '' } })).toBe('')
  })
})
