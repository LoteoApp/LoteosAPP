import { useState, type FormEvent } from 'react'
import { Link } from 'react-router'
import { messageFromError } from '../../../shared/api/client'
import { Button } from '../../../shared/ui/button'
import { Field, FieldError, FieldLabel } from '../../../shared/ui/field'
import { Input } from '../../../shared/ui/input'
import { requestPasswordReset } from '../api/auth'

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [sent, setSent] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setIsSubmitting(true)
    setError(null)

    try {
      await requestPasswordReset(email)
      setSent(true)
    } catch (requestError) {
      setError(messageFromError(requestError))
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-background px-4 py-10 text-foreground">
      <div className="w-full max-w-sm rounded-xl border border-border bg-card p-6 text-card-foreground md:p-8">
        <p className="text-sm font-medium uppercase tracking-[0.24em] text-primary">LoteosAPP</p>
        <h1 className="mt-3 text-2xl font-semibold tracking-tight">Olvidé mi contraseña</h1>

        {sent ? (
          <div className="mt-6 flex flex-col gap-4">
            <p className="text-sm text-muted-foreground">
              Si el correo está registrado, te enviamos un mail con un link para restablecer tu
              contraseña. Revisá tu bandeja de entrada.
            </p>
            <Link
              to="/login"
              className="text-center text-sm font-medium text-primary underline-offset-4 hover:underline"
            >
              Volver a iniciar sesión
            </Link>
          </div>
        ) : (
          <form className="mt-6 flex flex-col gap-4" onSubmit={handleSubmit}>
            <p className="text-sm text-muted-foreground">
              Ingresá tu correo electrónico y te vamos a mandar un link para elegir una contraseña
              nueva.
            </p>

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

            <FieldError>{error}</FieldError>

            <Button type="submit" size="lg" disabled={isSubmitting} className="mt-2 w-full">
              {isSubmitting ? 'Enviando...' : 'Enviar link'}
            </Button>

            <Link
              to="/login"
              className="text-center text-sm font-medium text-primary underline-offset-4 hover:underline"
            >
              Volver a iniciar sesión
            </Link>
          </form>
        )}
      </div>
    </main>
  )
}
