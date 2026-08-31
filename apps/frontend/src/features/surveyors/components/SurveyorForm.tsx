import { useState, type ChangeEvent, type FormEvent } from 'react'
import { Button } from '../../../shared/ui/button'
import { Input } from '../../../shared/ui/input'
import { Label } from '../../../shared/ui/label'
import type { SurveyorFormValues } from '../types'

const emptyValues: SurveyorFormValues = { nombre: '', apellido: '', email: '' }

type SurveyorFormProps = {
  initialValue?: SurveyorFormValues
  submitLabel: string
  isEmailEditable: boolean
  isSubmitting: boolean
  onSubmit: (values: SurveyorFormValues) => void
  onCancel: () => void
}

export default function SurveyorForm({
  initialValue,
  submitLabel,
  isEmailEditable,
  isSubmitting,
  onSubmit,
  onCancel,
}: SurveyorFormProps) {
  const [values, setValues] = useState<SurveyorFormValues>(initialValue ?? emptyValues)
  const [error, setError] = useState<string | null>(null)

  function handleChange(field: keyof SurveyorFormValues) {
    return (event: ChangeEvent<HTMLInputElement>) => {
      setValues((current) => ({ ...current, [field]: event.target.value }))
    }
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    const trimmed: SurveyorFormValues = {
      nombre: values.nombre.trim(),
      apellido: values.apellido.trim(),
      email: values.email.trim(),
    }

    if (!trimmed.nombre || !trimmed.apellido) {
      setError('Completá nombre y apellido.')
      return
    }
    if (isEmailEditable && !trimmed.email.includes('@')) {
      setError('Ingresá un correo electrónico válido.')
      return
    }

    setError(null)
    onSubmit(trimmed)
  }

  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={handleSubmit}
      noValidate
      aria-label="Datos del agrimensor"
    >
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="nombre">Nombre</Label>
          <Input id="nombre" name="nombre" value={values.nombre} onChange={handleChange('nombre')} />
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="apellido">Apellido</Label>
          <Input
            id="apellido"
            name="apellido"
            value={values.apellido}
            onChange={handleChange('apellido')}
          />
        </div>

        <div className="flex flex-col gap-1.5 sm:col-span-2">
          <Label htmlFor="email">Correo electrónico</Label>
          <Input
            id="email"
            name="email"
            type="email"
            value={values.email}
            disabled={!isEmailEditable}
            onChange={handleChange('email')}
          />
          {!isEmailEditable && (
            <p className="text-sm text-muted-foreground">
              El correo identifica la cuenta y no se puede modificar desde acá.
            </p>
          )}
        </div>
      </div>

      {error && (
        <p className="text-sm text-destructive" role="alert">
          {error}
        </p>
      )}

      <div className="flex flex-col gap-2 sm:flex-row sm:justify-end">
        <Button type="button" variant="outline" onClick={onCancel} disabled={isSubmitting}>
          Cancelar
        </Button>
        <Button type="submit" disabled={isSubmitting}>
          {submitLabel}
        </Button>
      </div>
    </form>
  )
}
