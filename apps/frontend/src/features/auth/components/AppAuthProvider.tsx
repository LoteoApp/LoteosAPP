import { useCallback, useEffect, useState, type ReactNode } from 'react'
import type { Session } from '@supabase/supabase-js'
import { supabaseClient } from '../config/supabase-client'
import { AuthContext, type AuthContextValue } from '../hooks/use-auth'

export default function AppAuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<Error | null>(null)

  useEffect(() => {
    const {
      data: { subscription },
    } = supabaseClient.auth.onAuthStateChange((_event, newSession) => {
      setSession(newSession)
      setIsLoading(false)
    })

    return () => subscription.unsubscribe()
  }, [])

  const login = useCallback(async (email: string, password: string) => {
    const { error: signInError } = await supabaseClient.auth.signInWithPassword({
      email,
      password,
    })
    if (signInError) {
      setError(signInError)
      throw signInError
    }
    setError(null)
  }, [])

  const logout = useCallback(async () => {
    const { error: signOutError } = await supabaseClient.auth.signOut()
    if (signOutError) {
      setError(signOutError)
      throw signOutError
    }
  }, [])

  const value: AuthContextValue = {
    isLoading,
    session,
    user: session?.user ?? null,
    error,
    login,
    logout,
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
