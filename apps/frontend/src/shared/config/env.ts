import { DEFAULT_API_URL, DEFAULT_SUPABASE_URL, resolveUrl } from './env-defaults'

export const apiUrl = resolveUrl(import.meta.env.VITE_API_URL, DEFAULT_API_URL)

export const supabaseUrl = resolveUrl(import.meta.env.VITE_SUPABASE_URL, DEFAULT_SUPABASE_URL)
export const supabaseAnonKey = import.meta.env.VITE_SUPABASE_ANON_KEY ?? ''

if (import.meta.env.PROD && !supabaseAnonKey) {
  throw new Error('VITE_SUPABASE_ANON_KEY must be set in production builds')
}
