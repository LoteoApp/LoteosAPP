import { createClient } from '@supabase/supabase-js'
import { supabaseAnonKey, supabaseUrl } from './env'

export const supabaseClient = createClient(supabaseUrl, supabaseAnonKey, {
  auth: { flowType: 'pkce' },
})
