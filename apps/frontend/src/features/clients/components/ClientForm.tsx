import { useState, type ChangeEvent, type FormEvent } from 'react'
import { Button } from '../../../shared/ui/button'
import type { ClienteFormValues } from '../types'

const fieldClassName =
  'h-10 w-full rounded-lg border border-border bg-background px-3 text-sm text-foreground outline-none transition-all placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50'

const emptyValues: ClienteFormValues = {
  nombre: '',
  apellido: '',
  dni: '',
  celular: '',
  email: '',
}

type ClientFormProps = {
  initialValue?: ClienteFormValues
  submitLabel: string
  onSubmit: (values: ClienteFormValues) => void
  onValidate: (values: ClienteFormValues) => string | null
  onCancel: () => void
}

export default function ClientForm({
  initialValue,
  submitLabel,
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
        <div className="flex flex-col gap-1.5">
          <label htmlFor="nombre" className="text-sm font-medium">
            Nombre
          </label>
          <input
            id="nombre"
            name="nombre"
            value={values.nombre}
            onChange={handleChange('nombre')}
            className={fieldClassName}
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <label htmlFor="apellido" className="text-sm font-medium">
            Apellido
          </label>
          <input
            id="apellido"
            name="apellido"
            value={values.apellido}
            onChange={handleChange('apellido')}
            className={fieldClassName}
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <label htmlFor="dni" className="text-sm font-medium">
            DNI
          </label>
          <input
            id="dni"
            name="dni"
            value={values.dni}
            onChange={handleChange('dni')}
            className={fieldClassName}
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <label htmlFor="celular" className="text-sm font-medium">
            Celular
          </label>
          <input
            id="celular"
            name="celular"
            type="tel"
            value={values.celular}
            onChange={handleChange('celular')}
            className={fieldClassName}
          />
        </div>

        <div className="flex flex-col gap-1.5 sm:col-span-2">
          <label htmlFor="email" className="text-sm font-medium">
            Correo electrónico
          </label>
          <input
            id="email"
            name="email"
            type="email"
            value={values.email}
            onChange={handleChange('email')}
            className={fieldClassName}
          />
        </div>
      </div>

      {error && (
        <p className="text-sm text-destructive" role="alert">
          {error}
        </p>
      )}

      <div className="flex flex-col gap-2 sm:flex-row sm:justify-end">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancelar
        </Button>
        <Button type="submit">{submitLabel}</Button>
      </div>
    </form>
  )
}
