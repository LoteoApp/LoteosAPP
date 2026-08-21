export const DEFAULT_API_URL = 'http://localhost:8080'
export const DEFAULT_SUPABASE_URL = 'http://127.0.0.1:54321'

export function resolveUrl(value: string | undefined, fallback: string): string {
  return (value ?? fallback).replace(/\/$/, '')
}
