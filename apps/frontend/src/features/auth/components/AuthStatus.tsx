import { useAuth } from 'react-oidc-context'
import { Button } from '../../../shared/ui/button'

export default function AuthStatus() {
  const auth = useAuth()

  if (auth.isLoading) {
    return <p className="text-sm text-muted-foreground">Verificando sesión...</p>
  }

  if (auth.error) {
    return (
      <div className="flex items-center gap-3 text-sm text-destructive" role="alert">
        <span>No se pudo validar la sesión: {auth.error.message}</span>
        <Button type="button" variant="outline" onClick={() => auth.signinRedirect()}>
          Reintentar
        </Button>
      </div>
    )
  }

  if (auth.isAuthenticated) {
    const displayName =
      auth.user?.profile.preferred_username ?? auth.user?.profile.name ?? auth.user?.profile.email

    return (
      <div className="flex items-center gap-3 text-sm">
        <span className="text-foreground">{displayName}</span>
        <Button type="button" variant="outline" onClick={() => auth.signoutRedirect()}>
          Cerrar sesión
        </Button>
      </div>
    )
  }

  return (
    <Button type="button" onClick={() => auth.signinRedirect()}>
      Iniciar sesión
    </Button>
  )
}
