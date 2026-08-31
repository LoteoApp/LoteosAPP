import { describe, expect, it } from 'vitest'
import { getUserRole, ROLE } from './roles'

describe('getUserRole', () => {
  it('returns the role stored in app_metadata', () => {
    expect(getUserRole({ app_metadata: { role: ROLE.administrador } })).toBe(
      ROLE.administrador,
    )
  })

  it('returns null when there is no user', () => {
    expect(getUserRole(null)).toBeNull()
    expect(getUserRole(undefined)).toBeNull()
  })

  it('returns null when the role is missing or empty', () => {
    expect(getUserRole({})).toBeNull()
    expect(getUserRole({ app_metadata: null })).toBeNull()
    expect(getUserRole({ app_metadata: { role: null } })).toBeNull()
    expect(getUserRole({ app_metadata: { role: '' } })).toBeNull()
  })
})
