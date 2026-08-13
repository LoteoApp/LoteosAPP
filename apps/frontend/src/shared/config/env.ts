export const apiUrl = (import.meta.env.VITE_API_URL ?? 'http://localhost:8080').replace(
  /\/$/,
  '',
)

export const supabaseUrl = (
  import.meta.env.VITE_SUPABASE_URL ?? 'http://127.0.0.1:54321'
).replace(/\/$/, '')
export const supabaseAnonKey = import.meta.env.VITE_SUPABASE_ANON_KEY ?? ''

if (import.meta.env.PROD && !supabaseAnonKey) {
  throw new Error('VITE_SUPABASE_ANON_KEY must be set in production builds')
}
