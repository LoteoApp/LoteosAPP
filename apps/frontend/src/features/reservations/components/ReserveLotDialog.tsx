import { CalendarPlus, X } from 'lucide-react'
import { useId, useState } from 'react'
import { Link } from 'react-router'
import { Button, buttonVariants } from '../../../shared/ui/button'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogTitle,
  DialogTrigger,
} from '../../../shared/ui/dialog'
import type { ReservationClient, ReservationDraft, ReservationLot } from '../types'
import ReservationForm from './ReservationForm'
import { useEligibleSellers } from '../hooks/use-eligible-sellers'
import { useReservationMutations } from '../hooks/use-reservation-mutations'
import { newIdempotencyKey } from '../lib/idempotencyKey'

type ReserveLotDialogProps = {
  accessToken: string
  loteoId: string
  lote: ReservationLot
  clients?: ReservationClient[]
  isLoadingClients?: boolean
  clientsError?: string | null
	onCreated: () => void
	navigationOnly?: boolean
}

function lotLabel(lote: ReservationLot): string {
  return lote.numero ? `Lote ${lote.numero}` : `Lote ${lote.id.slice(0, 8)}`
}

function reservationDisabledReason(lote: ReservationLot): string | null {
  const missing: string[] = []
  if (!lote.numero.trim()) {
    missing.push('el número')
  }
  if (lote.precio === null) {
    missing.push('el precio')
  }
  if (missing.length === 0) {
    return null
  }
  return `Completá ${missing.join(' y ')} del lote para habilitar la reserva.`
}

export default function ReserveLotDialog({
  accessToken,
  loteoId,
  lote,
  navigationOnly = false,
  onCreated,
  ...dialogProps
}: ReserveLotDialogProps) {
  const disabledReason = reservationDisabledReason(lote)
  if (navigationOnly) {
    if (disabledReason) {
      return (
        <span title={disabledReason} className="inline-flex">
          <Button type="button" disabled>
            <CalendarPlus aria-hidden />
            Reservar lote
          </Button>
        </span>
      )
    }
    return (
      <Link
        className={buttonVariants()}
        to={`/reservas/nueva/${encodeURIComponent(loteoId)}/${encodeURIComponent(lote.id)}`}
      >
        <CalendarPlus aria-hidden />
        Reservar lote
      </Link>
    )
  }
  return <ReservationDialog accessToken={accessToken} loteoId={loteoId} lote={lote} onCreated={onCreated} {...dialogProps} />
}

function ReservationDialog({
  accessToken,
  loteoId,
  lote,
  clients = [],
  isLoadingClients = false,
	clientsError = null,
	onCreated,
}: Omit<ReserveLotDialogProps, 'navigationOnly'>) {
  const [open, setOpen] = useState(false)
  const [idempotencyKey, setIdempotencyKey] = useState(newIdempotencyKey)
  const [draft, setDraft] = useState<ReservationDraft>({ clienteId: '', vendedorId: '' })
  const disabledTooltipId = useId()
  const { sellers, isLoading: isLoadingSellers, error: sellersError } = useEligibleSellers(
    accessToken,
    loteoId,
  )
  const mutations = useReservationMutations(accessToken)
  const disabledReason = reservationDisabledReason(lote)

  function handleOpenChange(nextOpen: boolean) {
    setOpen(nextOpen)
    if (!nextOpen) {
      mutations.reset()
    }
  }

  async function handleSubmit(
    values: { loteoId: string; loteId: string; clienteId: string; vendedorId?: string },
    key: string,
  ) {
    if (disabledReason) {
      return false
    }

    const created = await mutations.create(values, key)
    if (!created) {
      return false
    }

    onCreated()
    handleOpenChange(false)
    return true
  }

  const loadError = clientsError || sellersError
	const trigger = (
		<Button type="button" disabled={disabledReason !== null}>
      <CalendarPlus aria-hidden />
      Reservar lote
    </Button>
	)

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      {disabledReason ? (
        <span
          className="group/tooltip relative inline-flex"
          title={disabledReason}
          tabIndex={0}
          aria-describedby={disabledTooltipId}
        >
          {trigger}
          <span
            id={disabledTooltipId}
            role="tooltip"
            className="pointer-events-none absolute right-0 bottom-full z-20 mb-2 w-max max-w-[min(16rem,calc(100vw-2rem))] rounded-md bg-foreground px-2.5 py-1.5 text-center text-xs text-background opacity-0 shadow-lg transition-opacity group-hover/tooltip:opacity-100 group-focus/tooltip:opacity-100"
          >
            {disabledReason}
          </span>
        </span>
      ) : (
        <DialogTrigger render={trigger} />
      )}
      <DialogContent>
        <div className="flex items-start justify-between gap-4 border-b border-border px-5 py-4">
          <div className="grid gap-1">
            <DialogTitle>Nueva reserva · {lotLabel(lote)}</DialogTitle>
            <DialogDescription>
              Completá los datos para bloquear este lote durante 360 horas.
            </DialogDescription>
          </div>
          <DialogClose
            render={
              <Button type="button" variant="ghost" size="icon" aria-label="Cerrar reserva">
                <X aria-hidden />
              </Button>
            }
          />
        </div>
        <div className="grid gap-4 p-5">
          <div className="rounded-lg border border-lot-reserved-foreground/20 bg-lot-reserved/50 px-3 py-2">
            <p className="text-xs font-medium uppercase tracking-wide text-lot-reserved-foreground">
              Lote seleccionado
            </p>
            <p className="text-sm font-semibold">{lotLabel(lote)}</p>
            <p className="break-all text-xs text-muted-foreground">ID: {lote.id}</p>
          </div>
          {loadError && (
            <p className="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
              {loadError}
            </p>
          )}
          <ReservationForm
            loteos={[]}
            selectedLoteoId={loteoId}
            selectedLoteId={lote.id}
            lots={[lote]}
            clients={clients}
            sellers={sellers}
            isLoadingSellers={isLoadingSellers || isLoadingClients}
            isSubmitting={mutations.isSubmitting}
            error={mutations.error}
            fixedTarget
            idempotencyKey={idempotencyKey}
            onSubmit={handleSubmit}
            draft={draft}
            onDraftChange={setDraft}
            onCompleted={() => {
              setDraft({ clienteId: '', vendedorId: '' })
              setIdempotencyKey(newIdempotencyKey())
            }}
            onCancel={() => handleOpenChange(false)}
          />
        </div>
      </DialogContent>
    </Dialog>
  )
}
