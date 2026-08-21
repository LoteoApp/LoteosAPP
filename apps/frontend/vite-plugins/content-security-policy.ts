import type { Plugin } from 'vite'

export interface CspOptions {
  apiUrl: string
  supabaseUrl: string
  isDev: boolean
}

export function buildContentSecurityPolicy({ apiUrl, supabaseUrl, isDev }: CspOptions): string {
  const styleSrc = isDev ? `'self' 'unsafe-inline'` : `'self'`

  return [
    `default-src 'self'`,
    `script-src 'self'`,
    `style-src ${styleSrc}`,
    `img-src 'self' data:`,
    `font-src 'self'`,
    `connect-src 'self' ${supabaseUrl} ${apiUrl}`,
    `base-uri 'self'`,
    `form-action 'self'`,
    `object-src 'none'`,
  ].join('; ')
}

export function contentSecurityPolicyPlugin(options: CspOptions): Plugin {
  const content = buildContentSecurityPolicy(options)

  return {
    name: 'content-security-policy',
    transformIndexHtml() {
      return [
        {
          tag: 'meta',
          attrs: { 'http-equiv': 'Content-Security-Policy', content },
          injectTo: 'head',
        },
      ]
    },
  }
}
