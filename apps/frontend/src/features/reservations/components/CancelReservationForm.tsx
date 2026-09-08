import { useState, type FormEvent } from 'react'
import { Button } from '../../../shared/ui/button'
import { Field, FieldError, FieldLabel } from '../../../shared/ui/field'
import { Textarea } from '../../../shared/ui/textarea'

export default function CancelReservationForm({ isSubmitting, error, onSubmit }: {
  isSubmitting: boolean
  error: string | null
  onSubmit: (reason: string) => Promise<boolean>
}) {
  const [reason, setReason] = useState('')
  const [validationError, setValidationError] = useState<string | null>(null)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!reason.trim()) {
      setValidationError('La justificación es obligatoria.')
      return
    }
    if (await onSubmit(reason.trim())) setReason('')
  }

  return (
    <form className="grid gap-2" onSubmit={handleSubmit} noValidate>
      <Field data-invalid={Boolean(validationError || error)}>
        <FieldLabel htmlFor="razon-cancelacion">Justificación de la cancelación</FieldLabel>
        <Textarea
          id="razon-cancelacion"
          value={reason}
          maxLength={500}
          aria-invalid={Boolean(validationError || error)}
          aria-describedby={(validationError || error) ? 'razon-cancelacion-error' : undefined}
          onChange={(event) => { setReason(event.target.value); setValidationError(null) }}
          placeholder="Contá brevemente por qué se cancela"
        />
        <FieldError id="razon-cancelacion-error">{validationError ?? error}</FieldError>
      </Field>
      <Button type="submit" variant="destructive" disabled={isSubmitting}>{isSubmitting ? 'Cancelando…' : 'Cancelar reserva'}</Button>
    </form>
  )
}
