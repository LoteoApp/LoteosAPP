import { useState, type ChangeEvent, type FormEvent } from 'react'
import { Button } from '../../../shared/ui/button'
import { Input } from '../../../shared/ui/input'
import { Label } from '../../../shared/ui/label'
import { cn } from '../../../shared/lib/utils'
import type { ClienteFormValues } from '../types'

const emptyValues: ClienteFormValues = {
  nombre: '',
  apellido: '',
  dni: '',
  celular: '',
  email: '',
}

const FIELDS: ReadonlyArray<{
  name: keyof ClienteFormValues
  label: string
  type?: string
  full?: boolean
}> = [
  { name: 'nombre', label: 'Nombre' },
  { name: 'apellido', label: 'Apellido' },
  { name: 'dni', label: 'DNI' },
  { name: 'celular', label: 'Celular', type: 'tel' },
  { name: 'email', label: 'Correo electrónico', type: 'email', full: true },
]

type ClientFormProps = {
  initialValue?: ClienteFormValues
  submitLabel: string
  isSubmitting?: boolean
  onSubmit: (values: ClienteFormValues) => void
  onValidate: (values: ClienteFormValues) => string | null
  onCancel: () => void
}

export default function ClientForm({
  initialValue,
  submitLabel,
  isSubmitting = false,
  onSubmit,
  onValidate,
  onCancel,
}: ClientFormProps) {
  const [values, setValues] = useState<ClienteFormValues>(initialValue ?? emptyValues)
  const [error, setError] = useState<string | null>(null)

  function handleChange(field: keyof ClienteFormValues) {
    return (event: ChangeEvent<HTMLInputElement>) => {
      setValues((current) => ({ ...current, [field]: event.target.value }))
    }
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    if (isSubmitting) {
      return
    }

    const trimmed: ClienteFormValues = {
      nombre: values.nombre.trim(),
      apellido: values.apellido.trim(),
      dni: values.dni.trim(),
      celular: values.celular.trim(),
      email: values.email.trim(),
    }

    const validationError = onValidate(trimmed)
    if (validationError) {
      setError(validationError)
      return
    }

    setError(null)
    onSubmit(trimmed)
  }

  return (
    <form className="flex flex-col gap-4" onSubmit={handleSubmit} aria-label="Datos del cliente">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        {FIELDS.map((field) => (
          <div
            key={field.name}
            className={cn('flex flex-col gap-1.5', field.full && 'sm:col-span-2')}
          >
            <Label htmlFor={field.name}>{field.label}</Label>
            <Input
              id={field.name}
              name={field.name}
              type={field.type ?? 'text'}
              value={values[field.name]}
              onChange={handleChange(field.name)}
            />
          </div>
        ))}
      </div>

      {error && (
        <p className="text-sm text-destructive" role="alert">
          {error}
        </p>
      )}

      <div className="flex flex-col gap-2 sm:flex-row sm:justify-end">
        <Button type="button" variant="outline" disabled={isSubmitting} onClick={onCancel}>
          Cancelar
        </Button>
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting ? 'Guardando...' : submitLabel}
        </Button>
      </div>
    </form>
  )
}
