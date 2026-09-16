import { describe, expect, it } from 'vitest'
import { describeAuthError } from './describeAuthError'

describe('describeAuthError', () => {
  it('translates rejected credentials without revealing whether the account exists', () => {
    const message = describeAuthError({
      code: 'invalid_credentials',
      message: 'Invalid login credentials',
    })

    expect(message).toBe('Correo o contraseña incorrectos.')
  })

  it('translates an unconfirmed email address', () => {
    expect(describeAuthError({ code: 'email_not_confirmed' })).toBe(
      'Falta confirmar el correo electrónico de la cuenta.',
    )
  })

  it('translates too many attempts in a row', () => {
    expect(describeAuthError({ code: 'over_request_rate_limit' })).toBe(
      'Demasiados intentos seguidos. Esperá unos minutos y volvé a probar.',
    )
  })

  it('keeps the original message for an untranslated failure so it stays debuggable', () => {
    expect(describeAuthError(new Error('Failed to fetch'))).toBe(
      'No se pudo iniciar sesión: Failed to fetch',
    )
  })

  it('keeps the original message for an unknown error code', () => {
    const error = Object.assign(new Error('Database error querying schema'), {
      code: 'unexpected_failure',
    })

    expect(describeAuthError(error)).toBe(
      'No se pudo iniciar sesión: Database error querying schema',
    )
  })

  it('ignores a code that only matches an inherited object property', () => {
    expect(describeAuthError({ code: 'toString' })).toBe(
      'No se pudo iniciar sesión. Probá de nuevo en unos minutos.',
    )
  })

  it('falls back to a generic message when there is nothing to report', () => {
    expect(describeAuthError(undefined)).toBe(
      'No se pudo iniciar sesión. Probá de nuevo en unos minutos.',
    )
  })
})
