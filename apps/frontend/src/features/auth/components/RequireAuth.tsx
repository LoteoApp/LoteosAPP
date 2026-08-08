import { useEffect, type ReactNode } from 'react'
import { useAuth } from 'react-oidc-context'

export default function RequireAuth({ children }: { children: ReactNode }) {
  const auth = useAuth()

  useEffect(() => {
    if (!auth.isLoading && !auth.isAuthenticated && !auth.error) {
      auth.signinRedirect()
    }
  }, [auth])

  if (auth.error) {
    return (
      <p className="p-6 text-sm text-destructive" role="alert">
        No se pudo validar la sesión: {auth.error.message}
      </p>
    )
  }

  if (auth.isLoading) {
    return <p className="p-6 text-sm text-muted-foreground">Verificando sesión...</p>
  }

  if (!auth.isAuthenticated) {
    return <p className="p-6 text-sm text-muted-foreground">Redirigiendo al inicio de sesión...</p>
  }

  return children
}
