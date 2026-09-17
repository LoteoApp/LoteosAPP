import { useState, type FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router'
import { messageFromError } from '../../../shared/api/client'
import { Button } from '../../../shared/ui/button'
import { Field, FieldError, FieldLabel } from '../../../shared/ui/field'
import { Input } from '../../../shared/ui/input'
import { confirmPasswordReset } from '../api/auth'

const MIN_PASSWORD_LENGTH = 8

export default function ResetPasswordPage() {
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token') ?? ''
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [done, setDone] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError(null)

    if (password.length < MIN_PASSWORD_LENGTH) {
      setError(`La contraseña debe tener al menos ${MIN_PASSWORD_LENGTH} caracteres.`)
      return
    }
    if (password !== confirmPassword) {
      setError('Las contraseñas no coinciden.')
      return
    }

    setIsSubmitting(true)
    try {
      await confirmPasswordReset(token, password)
      setDone(true)
    } catch (confirmError) {
      setError(messageFromError(confirmError))
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-background px-4 py-10 text-foreground">
      <div className="w-full max-w-sm rounded-xl border border-border bg-card p-6 text-card-foreground md:p-8">
        <p className="text-sm font-medium uppercase tracking-[0.24em] text-primary">LoteosAPP</p>
        <h1 className="mt-3 text-2xl font-semibold tracking-tight">Restablecer contraseña</h1>

        {!token ? (
          <div className="mt-6 flex flex-col gap-4">
            <p className="text-sm text-destructive">
              El link para restablecer la contraseña es inválido o venció.
            </p>
            <Link
              to="/olvide-contrasena"
              className="text-center text-sm font-medium text-primary underline-offset-4 hover:underline"
            >
              Pedir un link nuevo
            </Link>
          </div>
        ) : done ? (
          <div className="mt-6 flex flex-col gap-4">
            <p className="text-sm text-muted-foreground">
              Tu contraseña se actualizó. Ya podés iniciar sesión con la contraseña nueva.
            </p>
            <Link
              to="/login"
              className="text-center text-sm font-medium text-primary underline-offset-4 hover:underline"
            >
              Iniciar sesión
            </Link>
          </div>
        ) : (
          <form className="mt-6 flex flex-col gap-4" onSubmit={handleSubmit}>
            <Field>
              <FieldLabel htmlFor="password">Contraseña nueva</FieldLabel>
              <Input
                id="password"
                type="password"
                name="password"
                autoComplete="new-password"
                required
                value={password}
                onChange={(event) => setPassword(event.target.value)}
              />
            </Field>

            <Field>
              <FieldLabel htmlFor="confirmPassword">Repetir contraseña</FieldLabel>
              <Input
                id="confirmPassword"
                type="password"
                name="confirmPassword"
                autoComplete="new-password"
                required
                value={confirmPassword}
                onChange={(event) => setConfirmPassword(event.target.value)}
              />
            </Field>

            <FieldError>{error}</FieldError>

            <Button type="submit" size="lg" disabled={isSubmitting} className="mt-2 w-full">
              {isSubmitting ? 'Guardando...' : 'Guardar contraseña'}
            </Button>
          </form>
        )}
      </div>
    </main>
  )
}
