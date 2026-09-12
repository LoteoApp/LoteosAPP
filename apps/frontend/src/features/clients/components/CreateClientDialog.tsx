import { useState, type ReactElement } from 'react'
import { X } from 'lucide-react'
import { Button } from '../../../shared/ui/button'
import { Alert, AlertDescription, AlertTitle } from '../../../shared/ui/alert'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogTitle,
  DialogTrigger,
} from '../../../shared/ui/dialog'
import ClientForm from './ClientForm'
import { createClient } from '../api/clients'
import type { Cliente, ClienteFormValues } from '../types'

type CreateClientDialogProps = {
  accessToken: string
  clients?: Cliente[]
  open: boolean
  onOpenChange: (open: boolean) => void
  onCreated: (client: Cliente) => void
  trigger?: ReactElement
}

export default function CreateClientDialog({
  accessToken,
  clients = [],
  open,
  onOpenChange,
  onCreated,
  trigger,
}: CreateClientDialogProps) {
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  function validate(values: ClienteFormValues): string | null {
    if (!values.nombre || !values.apellido || !values.dni) {
      return 'Completá nombre, apellido y DNI.'
    }
    if (clients.some((client) => client.dni === values.dni)) {
      return 'Ya existe un cliente con ese DNI.'
    }
    return null
  }

  async function handleSubmit(values: ClienteFormValues) {
    setIsSubmitting(true)
    setError(null)
    try {
      const client = await createClient(accessToken, values)
      onCreated(client)
      onOpenChange(false)
    } catch (createError) {
      setError(createError instanceof Error ? createError.message : 'No se pudo crear el cliente.')
    } finally {
      setIsSubmitting(false)
    }
  }

  function handleOpenChange(nextOpen: boolean) {
    if (nextOpen) {
      setError(null)
    }
    onOpenChange(nextOpen)
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      {trigger && <DialogTrigger render={trigger} />}
      <DialogContent>
        <div className="flex items-start justify-between gap-4 border-b border-border px-5 py-4">
          <div className="grid gap-1">
            <DialogTitle>Nuevo cliente</DialogTitle>
            <DialogDescription>Registrá los datos para asociarlo a la reserva.</DialogDescription>
          </div>
          <DialogClose
            render={
              <Button type="button" variant="ghost" size="icon" aria-label="Cerrar alta de cliente">
                <X aria-hidden />
              </Button>
            }
          />
        </div>
        <div className="grid gap-4 p-5">
          {error && (
            <Alert variant="destructive">
              <AlertTitle>No se pudo crear el cliente</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
          <ClientForm
            key={open ? 'open' : 'closed'}
            submitLabel="Crear cliente"
            isSubmitting={isSubmitting}
            onSubmit={handleSubmit}
            onValidate={validate}
            onCancel={() => onOpenChange(false)}
          />
        </div>
      </DialogContent>
    </Dialog>
  )
}
