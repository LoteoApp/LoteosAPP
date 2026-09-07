import { X } from 'lucide-react'
import { Button } from '../../../shared/ui/button'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from '../../../shared/ui/dialog'
import type { Reservation } from '../types'
import CancelReservationForm from './CancelReservationForm'

type CancelReservationDialogProps = {
  reservation: Reservation
  isSubmitting: boolean
  error: string | null
  onSubmit: (reason: string) => Promise<boolean>
  onClose: () => void
}

function lotLabel(reservation: Reservation): string {
  return reservation.loteNumero ? `Lote ${reservation.loteNumero}` : `Lote ${reservation.loteId.slice(0, 8)}`
}

export default function CancelReservationDialog({
  reservation,
  isSubmitting,
  error,
  onSubmit,
  onClose,
}: CancelReservationDialogProps) {
  async function handleSubmit(reason: string): Promise<boolean> {
    const completed = await onSubmit(reason)
    if (completed) onClose()
    return completed
  }

  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent>
        <div className="flex items-start justify-between gap-4 border-b border-border px-5 py-4">
          <div className="grid gap-1">
            <DialogTitle>Cancelar reserva</DialogTitle>
            <DialogDescription>Esta acción libera el lote antes de su vencimiento.</DialogDescription>
          </div>
          <DialogClose
            render={
              <Button type="button" variant="ghost" size="icon" aria-label="Cerrar cancelación">
                <X aria-hidden />
              </Button>
            }
          />
        </div>
        <div className="grid gap-4 p-5">
          <div className="rounded-lg border border-border bg-muted/30 px-3 py-2">
            <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Reserva seleccionada</p>
            <p className="text-sm font-semibold">{reservation.loteoNombre} · {lotLabel(reservation)}</p>
            <p className="text-sm text-muted-foreground">Cliente: {reservation.cliente.nombre} {reservation.cliente.apellido}</p>
            <p className="break-all text-xs text-muted-foreground">ID: {reservation.id}</p>
          </div>
          <CancelReservationForm isSubmitting={isSubmitting} error={error} onSubmit={handleSubmit} />
        </div>
      </DialogContent>
    </Dialog>
  )
}
