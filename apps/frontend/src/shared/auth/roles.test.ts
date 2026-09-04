import { describe, expect, it } from 'vitest'
import { getUserRole } from './roles'

describe('roles', () => {
  it('reads the domain role from app_metadata', () => {
    expect(getUserRole({ app_metadata: { role: 'administrador' } })).toBe('administrador')
  })

  it('returns null when app_metadata has no role', () => {
    expect(getUserRole({ app_metadata: {} })).toBeNull()
  })

  it('returns null when app_metadata is missing', () => {
    expect(getUserRole({})).toBeNull()
  })

  it('returns null when there is no user', () => {
    expect(getUserRole(null)).toBeNull()
  })
})
