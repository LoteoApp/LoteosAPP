import { createClient } from '@supabase/supabase-js'
import { describe, expect, it, vi } from 'vitest'
import { supabaseAnonKey, supabaseUrl } from './env'
import './supabase-client'

vi.mock('@supabase/supabase-js', () => ({
  createClient: vi.fn(() => ({})),
}))

describe('supabaseClient', () => {
  it('creates the Supabase client with the app URL, anon key, and the PKCE auth flow', () => {
    expect(createClient).toHaveBeenCalledWith(supabaseUrl, supabaseAnonKey, {
      auth: { flowType: 'pkce' },
    })
  })
})
