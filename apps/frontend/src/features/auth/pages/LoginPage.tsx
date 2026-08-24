import { useState, type FormEvent } from 'react'
import { Navigate, useLocation, useNavigate } from 'react-router'
import { Button } from '../../../shared/ui/button'
import { Field, FieldError, FieldLabel } from '../../../shared/ui/field'
import { Input } from '../../../shared/ui/input'
import { useAuth } from '../hooks/use-auth'
import { describeAuthError } from '../lib/describeAuthError'

// A "from" starting with "//" is resolved by the browser as an external host.
function resolveRedirectPath(state: unknown): string {
  const from = (state as { from?: unknown } | null)?.from

  if (typeof from === 'string' && from.startsWith('/') && !from.startsWith('//')) {
    return from
  }

  return '/lotes'
}

export default function LoginPage() {
  const { isLoading, session, login } = useAuth()
  const location = useLocation()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const redirectTo = resolveRedirectPath(location.state)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setIsSubmitting(true)
    setError(null)

    try {
      await login(email, password)
      await navigate(redirectTo, { replace: true })
    } catch (loginError) {
      setError(describeAuthError(loginError))
    } finally {
      setIsSubmitting(false)
    }
  }

  if (isLoading) {
    return <p className="p-6 text-sm text-muted-foreground">Verificando sesión...</p>
  }

  if (session) {
    return <Navigate to={redirectTo} replace />
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-background px-4 py-10 text-foreground">
      <div className="w-full max-w-sm rounded-xl border border-border bg-card p-6 text-card-foreground md:p-8">
        <p className="text-sm font-medium uppercase tracking-[0.24em] text-primary">
          LoteosAPP
        </p>
        <h1 className="mt-3 text-2xl font-semibold tracking-tight">Iniciar sesión</h1>

        <form className="mt-6 flex flex-col gap-4" onSubmit={handleSubmit}>
          <Field>
            <FieldLabel htmlFor="email">Correo electrónico</FieldLabel>
            <Input
              id="email"
              type="email"
              name="email"
              autoComplete="email"
              required
              value={email}
              onChange={(event) => setEmail(event.target.value)}
            />
          </Field>

          <Field>
            <FieldLabel htmlFor="password">Contraseña</FieldLabel>
            <Input
              id="password"
              type="password"
              name="password"
              autoComplete="current-password"
              required
              value={password}
              onChange={(event) => setPassword(event.target.value)}
            />
          </Field>

          <FieldError>{error}</FieldError>

          <Button type="submit" size="lg" disabled={isSubmitting} className="mt-2 w-full">
            {isSubmitting ? 'Ingresando...' : 'Ingresar'}
          </Button>
        </form>
      </div>
    </main>
  )
}
