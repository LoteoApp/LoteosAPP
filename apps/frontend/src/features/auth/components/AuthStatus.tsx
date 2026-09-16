import { Link } from 'react-router'
import { Button, buttonVariants } from '../../../shared/ui/button'
import { useAuth } from '../hooks/use-auth'
import { resolveDisplayName } from '../lib/resolveDisplayName'

export default function AuthStatus() {
  const { isLoading, user, error, logout } = useAuth()

  if (isLoading) {
    return <p className="text-sm text-muted-foreground">Verificando sesión...</p>
  }

  if (error) {
    return (
      <div className="flex items-center gap-3 text-sm text-destructive" role="alert">
        <span>No se pudo validar la sesión: {error.message}</span>
        <Link to="/login" className={buttonVariants({ variant: 'outline' })}>
          Reintentar
        </Link>
      </div>
    )
  }

  if (user) {
    return (
      <div className="flex items-center gap-3 text-sm">
        <span className="text-foreground">{resolveDisplayName(user)}</span>
        <Button
          type="button"
          variant="outline"
          onClick={() => {
            logout().catch(() => {})
          }}
        >
          Cerrar sesión
        </Button>
      </div>
    )
  }

  return (
    <Link to="/login" className={buttonVariants()}>
      Iniciar sesión
    </Link>
  )
}
