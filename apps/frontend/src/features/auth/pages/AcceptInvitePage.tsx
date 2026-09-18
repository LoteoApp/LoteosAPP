import { useEffect, useState, type FormEvent } from 'react'
import { Link } from 'react-router'
import type { EmailOtpType } from '@supabase/supabase-js'
import { supabaseClient } from '../../../shared/config/supabase-client'
import { clearURLFragment, readFragmentParams } from '../../../shared/lib/urlFragment'
import { Button } from '../../../shared/ui/button'
import { Field, FieldError, FieldLabel } from '../../../shared/ui/field'
import { Input } from '../../../shared/ui/input'

const MIN_PASSWORD_LENGTH = 8

function describeInviteError(error: unknown): string {
  const code = error && typeof error === 'object' ? (error as { code?: unknown }).code : undefined
  if (code === 'otp_expired') {
    return 'El link de invitación venció. Pedile a un administrador que te reenvíe uno nuevo.'
  }
  if (error instanceof Error && error.message !== '') {
    return `No se pudo activar la cuenta: ${error.message}`
  }
  return 'No se pudo activar la cuenta. Probá de nuevo en unos minutos.'
}

export default function AcceptInvitePage() {
  const [fragmentParams] = useState(() => readFragmentParams())
  useEffect(() => {
    clearURLFragment()
  }, [])
  const tokenHash = fragmentParams.get('token_hash') ?? ''
  const type = (fragmentParams.get('type') ?? 'invite') as EmailOtpType
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
      const { error: verifyError } = await supabaseClient.auth.verifyOtp({ token_hash: tokenHash, type })
      if (verifyError) {
        throw verifyError
      }

      const { error: updateError } = await supabaseClient.auth.updateUser({ password })
      if (updateError) {
        throw updateError
      }

      await supabaseClient.auth.signOut()
      setDone(true)
    } catch (acceptError) {
      setError(describeInviteError(acceptError))
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-background px-4 py-10 text-foreground">
      <div className="w-full max-w-sm rounded-xl border border-border bg-card p-6 text-card-foreground md:p-8">
        <p className="text-sm font-medium uppercase tracking-[0.24em] text-primary">LoteosAPP</p>
        <h1 className="mt-3 text-2xl font-semibold tracking-tight">Activar cuenta</h1>

        {!tokenHash ? (
          <div className="mt-6 flex flex-col gap-4">
            <p className="text-sm text-destructive">El link de invitación es inválido o venció.</p>
          </div>
        ) : done ? (
          <div className="mt-6 flex flex-col gap-4">
            <p className="text-sm text-muted-foreground">
              Tu cuenta quedó activada. Ya podés iniciar sesión con la contraseña que elegiste.
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
              <FieldLabel htmlFor="password">Elegí tu contraseña</FieldLabel>
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
              {isSubmitting ? 'Activando...' : 'Activar cuenta'}
            </Button>
          </form>
        )}
      </div>
    </main>
  )
}
