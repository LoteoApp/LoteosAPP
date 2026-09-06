import { useState, type ChangeEvent, type FormEvent } from 'react'
import { Dialog } from '@base-ui/react/dialog'
import { Button } from '../../../shared/ui/button'
import { Input } from '../../../shared/ui/input'
import { Label } from '../../../shared/ui/label'
import { cn } from '../../../shared/lib/utils'
import type { NewClientValues } from '../types'

const emptyValues: NewClientValues = {
  nombre: '',
  apellido: '',
  dni: '',
  celular: '',
  email: '',
}

const FIELDS: ReadonlyArray<{
  name: keyof NewClientValues
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

type NewClientDialogProps = {
  open: boolean
  isSubmitting?: boolean
  error?: string | null
  onSubmit: (values: NewClientValues) => void
  onClose: () => void
}

export default function NewClientDialog({
  open,
  isSubmitting = false,
  error = null,
  onSubmit,
  onClose,
}: NewClientDialogProps) {
  const [values, setValues] = useState<NewClientValues>(emptyValues)
  const [validationError, setValidationError] = useState<string | null>(null)

  function handleOpenChange(next: boolean) {
    if (next) {
      return
    }
    setValues(emptyValues)
    setValidationError(null)
    onClose()
  }

  function handleChange(field: keyof NewClientValues) {
    return (event: ChangeEvent<HTMLInputElement>) => {
      setValues((current) => ({ ...current, [field]: event.target.value }))
    }
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    if (isSubmitting) {
      return
    }

    const trimmed: NewClientValues = {
      nombre: values.nombre.trim(),
      apellido: values.apellido.trim(),
      dni: values.dni.trim(),
      celular: values.celular.trim(),
      email: values.email.trim(),
    }

    if (!trimmed.nombre || !trimmed.apellido || !trimmed.dni) {
      setValidationError('Completá nombre, apellido y DNI.')
      return
    }

    setValidationError(null)
    onSubmit(trimmed)
  }

  const message = validationError ?? error

  return (
    <Dialog.Root open={open} onOpenChange={handleOpenChange}>
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-50 bg-black/50" />
        <Dialog.Popup className="fixed top-1/2 left-1/2 z-50 max-h-[90dvh] w-[calc(100vw-2rem)] max-w-lg -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-lg border border-border bg-card p-6 shadow-lg">
          <Dialog.Title className="text-lg font-semibold text-foreground">
            Registrar cliente
          </Dialog.Title>
          <Dialog.Description className="mt-1 text-sm text-muted-foreground">
            El cliente queda disponible para seleccionarlo en la venta.
          </Dialog.Description>

          <form className="mt-4 flex flex-col gap-4" onSubmit={handleSubmit} noValidate>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              {FIELDS.map((field) => (
                <div
                  key={field.name}
                  className={cn('flex flex-col gap-1.5', field.full && 'sm:col-span-2')}
                >
                  <Label htmlFor={`nuevo-cliente-${field.name}`}>{field.label}</Label>
                  <Input
                    id={`nuevo-cliente-${field.name}`}
                    type={field.type ?? 'text'}
                    value={values[field.name]}
                    onChange={handleChange(field.name)}
                    disabled={isSubmitting}
                  />
                </div>
              ))}
            </div>

            {message !== null && message !== '' && (
              <p role="alert" className="text-sm text-destructive">
                {message}
              </p>
            )}

            <div className="flex flex-col gap-2 sm:flex-row sm:justify-end">
              <Button
                type="button"
                variant="outline"
                onClick={() => handleOpenChange(false)}
                disabled={isSubmitting}
              >
                Cancelar
              </Button>
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting ? 'Guardando…' : 'Guardar cliente'}
              </Button>
            </div>
          </form>
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
