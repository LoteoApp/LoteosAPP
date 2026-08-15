import { DEFAULT_API_URL, DEFAULT_SUPABASE_URL } from './env-defaults'

export const apiUrl = (import.meta.env.VITE_API_URL ?? DEFAULT_API_URL).replace(/\/$/, '')

export const supabaseUrl = (import.meta.env.VITE_SUPABASE_URL ?? DEFAULT_SUPABASE_URL).replace(
  /\/$/,
  '',
)
export const supabaseAnonKey = import.meta.env.VITE_SUPABASE_ANON_KEY ?? ''

if (import.meta.env.PROD && !supabaseAnonKey) {
  throw new Error('VITE_SUPABASE_ANON_KEY must be set in production builds')
}
