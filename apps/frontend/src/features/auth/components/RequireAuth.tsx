import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router'
import { useAuth } from '../hooks/use-auth'

export default function RequireAuth({ children }: { children: ReactNode }) {
  const { isLoading, session } = useAuth()
  const location = useLocation()

  if (isLoading) {
    return <p className="p-6 text-sm text-muted-foreground">Verificando sesión...</p>
  }

  if (!session) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />
  }

  return children
}
