import { describe, expect, it } from 'vitest'
import { buildContentSecurityPolicy } from './content-security-policy'

describe('buildContentSecurityPolicy', () => {
  it('scopes connect-src to the exact backend and Supabase origins, never a wildcard', () => {
    const csp = buildContentSecurityPolicy({
      apiUrl: 'http://localhost:8080',
      supabaseUrl: 'https://iahqjtpzkntzxoiykhjg.supabase.co',
      isDev: false,
    })

    expect(csp).toContain(
      "connect-src 'self' https://iahqjtpzkntzxoiykhjg.supabase.co http://localhost:8080",
    )
    expect(csp).not.toContain('*.supabase.co')
  })

  it('never leaves an unresolved placeholder in the policy', () => {
    const csp = buildContentSecurityPolicy({
      apiUrl: 'http://localhost:8080',
      supabaseUrl: 'https://iahqjtpzkntzxoiykhjg.supabase.co',
      isDev: false,
    })

    expect(csp).not.toMatch(/%VITE_\w+%/)
  })

  it('allows inline styles only in dev, for Vite/Tailwind CSS injection', () => {
    const dev = buildContentSecurityPolicy({
      apiUrl: 'http://localhost:8080',
      supabaseUrl: 'https://example.supabase.co',
      isDev: true,
    })
    const prod = buildContentSecurityPolicy({
      apiUrl: 'http://localhost:8080',
      supabaseUrl: 'https://example.supabase.co',
      isDev: false,
    })

    expect(dev).toContain("style-src 'self' 'unsafe-inline'")
    expect(prod).toContain("style-src 'self'")
    expect(prod).not.toContain('unsafe-inline')
  })

  it('never allows unsafe-eval or inline scripts, in dev or prod', () => {
    for (const isDev of [true, false]) {
      const csp = buildContentSecurityPolicy({
        apiUrl: 'http://localhost:8080',
        supabaseUrl: 'https://example.supabase.co',
        isDev,
      })

      expect(csp).toContain("script-src 'self'")
      expect(csp).not.toContain('unsafe-eval')
    }
  })
})
