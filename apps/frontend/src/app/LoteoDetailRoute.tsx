import { useState } from 'react'
import { Clock, ExternalLink, X } from 'lucide-react'
import { Link } from 'react-router'
import { formatDateTime } from '../shared/lib/formatDateTime'
import { useAuth } from '../features/auth/hooks/use-auth'
import { getUserRole, ROLE } from '../shared/auth/roles'
import { Alert, AlertDescription, AlertTitle } from '../shared/ui/alert'
import { Button } from '../shared/ui/button'
import LoteoDetailPage from '../features/lots/pages/LoteoDetailPage'
import CancelReservationDialog from '../features/reservations/components/CancelReservationDialog'
import ReservationsList from '../features/reservations/components/ReservationsList'
import ReservationsPagination from '../features/reservations/components/ReservationsPagination'
import ReserveLotDialog from '../features/reservations/components/ReserveLotDialog'
import { useReservationMutations } from '../features/reservations/hooks/use-reservation-mutations'
import { useReservations } from '../features/reservations/hooks/use-reservations'
import type { Reservation } from '../features/reservations/types'
import type { LoteoDetail, LoteoLote } from '../features/lots/types'
import { useParams } from 'react-router'

export default function LoteoDetailRoute() {
  const { session, user } = useAuth()
  const { loteoId = '' } = useParams()
  const role = getUserRole(user ?? session?.user)
  const canEdit = role === ROLE.administrador || role === ROLE.agrimensor
  const canReserve = role === ROLE.administrador || role === ROLE.administrativo || role === ROLE.inmobiliaria
  const token = session?.access_token ?? ''
  const mutations = useReservationMutations(token)
  const [cancelTarget, setCancelTarget] = useState<{
    reservation: Reservation
    onCanceled: () => void
    refresh: () => void
  } | null>(null)

  const renderReservationAction = canReserve
    ? (lote: LoteoLote, onCreated: () => void) => (
        <ReserveLotDialog
          accessToken={session?.access_token ?? ''}
          loteoId={loteoId}
          lote={lote}
          navigationOnly
          onCreated={onCreated}
        />
      )
    : undefined

  const renderReservationSummary = canReserve
    ? (lote: LoteoLote, onCanceled: () => void) => (
        <LotReservationSummary
          token={token}
          loteoId={loteoId}
          lote={lote}
          onCanceled={onCanceled}
          onSelect={setCancelTarget}
        />
      )
    : undefined

  const renderReservations = canReserve
    ? (loteo: LoteoDetail, onCanceled: (loteId: string) => void) => (
        <LoteoReservations
          key={loteo.id}
          token={token}
          loteoId={loteo.id}
          onCanceled={onCanceled}
          onSelect={setCancelTarget}
        />
      )
    : undefined

  async function handleCancel(reason: string) {
    if (!cancelTarget) return false
    const canceled = await mutations.cancel(cancelTarget.reservation.id, reason)
    if (!canceled) return false
    cancelTarget.onCanceled()
    setCancelTarget(null)
    cancelTarget.refresh()
    return true
  }

  function closeCancelDialog() {
    setCancelTarget(null)
    mutations.reset()
  }

  return (
    <>
      <LoteoDetailPage
        accessToken={token || null}
        canEdit={canEdit}
        renderReservationAction={renderReservationAction}
        renderReservationSummary={renderReservationSummary}
        renderReservations={renderReservations}
      />
      {cancelTarget && (
        <CancelReservationDialog
          reservation={cancelTarget.reservation}
          isSubmitting={mutations.isSubmitting}
          error={mutations.error}
          onSubmit={handleCancel}
          onClose={closeCancelDialog}
        />
      )}
    </>
  )
}

function LoteoReservations({
  token,
  loteoId,
  onCanceled,
  onSelect,
}: {
  token: string
  loteoId: string
  onCanceled: (loteId: string) => void
  onSelect: (target: { reservation: Reservation; onCanceled: () => void; refresh: () => void }) => void
}) {
  const [pageNumber, setPageNumber] = useState(1)
  const reservations = useReservations(
    token,
    { loteoId, estado: 'activa', pagina: pageNumber },
    { enabled: token !== '' },
  )

  if (reservations.isLoading) {
    return <p className="text-sm text-muted-foreground">Cargando las reservas…</p>
  }

  if (reservations.error) {
    return (
      <Alert variant="destructive">
        <AlertTitle>No se pudieron cargar las reservas</AlertTitle>
        <AlertDescription>{reservations.error}</AlertDescription>
      </Alert>
    )
  }

  return (
    <div className="flex flex-col gap-3">
      <ReservationsList
        reservations={reservations.page.reservas}
        onCancel={(reservation) =>
          onSelect({
            reservation,
            onCanceled: () => onCanceled(reservation.loteId),
            refresh: reservations.refresh,
          })
        }
      />
      <ReservationsPagination
        page={reservations.page}
        isLoading={reservations.isLoading}
        onPageChange={setPageNumber}
      />
    </div>
  )
}

function LotReservationSummary({
  token,
  loteoId,
  lote,
  onCanceled,
  onSelect,
}: {
  token: string
  loteoId: string
  lote: LoteoLote
  onCanceled: () => void
  onSelect: (target: { reservation: Reservation; onCanceled: () => void; refresh: () => void }) => void
}) {
  const enabled = token !== '' && lote.estado === 'reservado'
  const reservations = useReservations(
    token,
    { loteoId, loteId: lote.id, estado: 'activa', porPagina: 1 },
    { enabled },
  )

  if (!enabled || reservations.isLoading || reservations.error) {
    return null
  }

  const reservation = reservations.page.reservas[0]
  if (!reservation) {
    return null
  }

  return (
    <div className="flex flex-col gap-2 rounded-lg border border-lot-reserved-foreground/20 bg-lot-reserved p-3 text-sm text-lot-reserved-foreground">
      <p className="flex items-center gap-2">
        <Clock aria-hidden className="size-4 shrink-0" />
        <span>
          Reservado por {reservation.cliente.nombre} {reservation.cliente.apellido} · vence el{' '}
          {formatDateTime(reservation.fechaVencimiento)}
        </span>
      </p>
      <div className="flex flex-wrap gap-2">
        {reservation.puedeCancelar === true && <Button
            type="button"
            variant="outline"
            onClick={() => onSelect({ reservation, onCanceled, refresh: reservations.refresh })}
          >
            <X aria-hidden />
            Cancelar reserva
          </Button>}
        <Link
          to={`/reservas/${reservation.id}`}
          className="inline-flex min-h-9 items-center gap-1.5 rounded-md border border-lot-reserved-foreground/30 px-3 text-sm font-medium hover:bg-lot-reserved-foreground/10"
        >
          <ExternalLink aria-hidden className="size-4" />
          Ver reserva
        </Link>
      </div>
    </div>
  )
}
