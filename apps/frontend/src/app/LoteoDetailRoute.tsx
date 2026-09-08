import { useState } from 'react'
import { useAuth } from '../features/auth/hooks/use-auth'
import { useClients } from '../features/clients/hooks/use-clients'
import { getUserRole, ROLE } from '../shared/auth/roles'
import { Button } from '../shared/ui/button'
import LoteoDetailPage from '../features/lots/pages/LoteoDetailPage'
import CancelReservationDialog from '../features/reservations/components/CancelReservationDialog'
import ReserveLotDialog from '../features/reservations/components/ReserveLotDialog'
import { useReservationMutations } from '../features/reservations/hooks/use-reservation-mutations'
import { useReservations } from '../features/reservations/hooks/use-reservations'
import type { Reservation } from '../features/reservations/types'
import type { LoteoLote } from '../features/lots/types'
import { useParams } from 'react-router'

export default function LoteoDetailRoute() {
  const { session, user } = useAuth()
  const { loteoId = '' } = useParams()
  const role = getUserRole(user ?? session?.user)
  const canEdit = role === ROLE.administrador || role === ROLE.agrimensor
  const canReserve = role === ROLE.administrador || role === ROLE.administrativo || role === ROLE.inmobiliaria
  const canCancel = canReserve
  const token = session?.access_token ?? ''
  const clients = useClients(token, { enabled: canReserve })
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
          clients={clients.clientes}
          isLoadingClients={clients.isLoading}
          clientsError={clients.error}
          onCreated={onCreated}
        />
      )
    : undefined

  const renderCancellationAction = canCancel
    ? (lote: LoteoLote, onCanceled: () => void) => {
        return <LotReservationCancelAction token={token} loteoId={loteoId} lote={lote} onCanceled={onCanceled} onSelect={setCancelTarget} />
      }
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
        renderReservationCancelAction={renderCancellationAction}
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

function LotReservationCancelAction({
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
  const reservations = useReservations(
    token,
    { loteoId, loteId: lote.id, estado: 'activa', porPagina: 1 },
    { enabled: token !== '' },
  )
  const reservation = reservations.page.reservas[0]
  if (reservations.isLoading || reservations.error || !reservation) return null
  return (
    <Button type="button" variant="outline" onClick={() => onSelect({ reservation, onCanceled, refresh: reservations.refresh })}>
      Cancelar reserva
    </Button>
  )
}
