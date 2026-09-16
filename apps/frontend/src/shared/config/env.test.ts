import { afterEach, describe, expect, it, vi } from 'vitest'

describe('env', () => {
  afterEach(() => {
    vi.unstubAllEnvs()
    vi.resetModules()
  })

  it('falls back to the local Supabase URL when unset', async () => {
    vi.stubEnv('VITE_SUPABASE_URL', undefined)

    const { supabaseUrl } = await import('./env')

    expect(supabaseUrl).toBe('http://127.0.0.1:54321')
  })

  it('throws when the anon key is missing in a production build', async () => {
    vi.stubEnv('VITE_SUPABASE_ANON_KEY', undefined)
    vi.stubEnv('PROD', true)

    await expect(import('./env')).rejects.toThrow(
      'VITE_SUPABASE_ANON_KEY must be set in production builds',
    )
  })

  it('does not throw outside production when the anon key is missing', async () => {
    vi.stubEnv('VITE_SUPABASE_ANON_KEY', undefined)
    vi.stubEnv('PROD', false)

    await expect(import('./env')).resolves.toBeDefined()
  })
})
