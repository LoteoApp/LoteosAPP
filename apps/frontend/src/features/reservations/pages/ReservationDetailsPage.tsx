import { useEffect, useRef, useState } from 'react'
import { Link, useParams } from 'react-router'
import { Alert, AlertDescription, AlertTitle } from '../../../shared/ui/alert'
import { Button } from '../../../shared/ui/button'
import { getReservation } from '../api/reservations'
import CancelReservationDialog from '../components/CancelReservationDialog'
import ReservationDetails from '../components/ReservationDetails'
import { useReservationMutations } from '../hooks/use-reservation-mutations'
import type { Reservation } from '../types'

type ReservationDetailsPageProps = { accessToken?: string }

export default function ReservationDetailsPage({ accessToken = '' }: ReservationDetailsPageProps) {
  const token = accessToken
  const { id = '' } = useParams()
  const [reservation, setReservation] = useState<Reservation | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [isCancelDialogOpen, setIsCancelDialogOpen] = useState(false)
  const mutations = useReservationMutations(token)
  const latestRequestRef = useRef({ id, token })
  const requestKey = JSON.stringify([id, token])
  const [loadedKey, setLoadedKey] = useState(requestKey)
  if (requestKey !== loadedKey) {
    setLoadedKey(requestKey)
    setReservation(null)
    setIsLoading(true)
    setError(null)
    setIsCancelDialogOpen(false)
  }

  useEffect(() => {
    latestRequestRef.current = { id, token }
  }, [id, token])

  useEffect(() => {
    const controller = new AbortController()
    getReservation(token, id, controller.signal)
      .then((loaded) => { if (!controller.signal.aborted) { setReservation(loaded); setError(null) } })
      .catch((loadError: unknown) => { if (!controller.signal.aborted) setError(loadError instanceof Error ? loadError.message : 'No se pudo cargar la reserva.') })
      .finally(() => { if (!controller.signal.aborted) setIsLoading(false) })
    return () => controller.abort()
  }, [id, token])

  async function handleCancel(reason: string) {
    if (!reservation) return false
    const targetID = reservation.id
    const targetToken = token
    const updated = await mutations.cancel(targetID, reason)
    if (!updated) return false
    if (latestRequestRef.current.id !== targetID || latestRequestRef.current.token !== targetToken) return false
    setReservation(updated)
    setIsCancelDialogOpen(false)
    return true
  }

  function closeCancelDialog() {
    setIsCancelDialogOpen(false)
    mutations.reset()
  }

  return (
    <section className="flex flex-col gap-4">
      <Link to="/reservas" className="w-fit text-sm text-muted-foreground hover:text-foreground">Volver a reservas</Link>
      {isLoading && <p className="text-muted-foreground">Cargando reserva…</p>}
      {error && <Alert variant="destructive"><AlertTitle>No se pudo cargar la reserva</AlertTitle><AlertDescription>{error}</AlertDescription></Alert>}
      {!isLoading && !error && reservation && <>
        <ReservationDetails reservation={reservation} />
        {reservation.estado === 'activa' && <>
          <Button variant="outline" className="w-fit" onClick={() => { mutations.reset(); setIsCancelDialogOpen(true) }}>Cancelar</Button>
          {isCancelDialogOpen && <CancelReservationDialog reservation={reservation} isSubmitting={mutations.isSubmitting} error={mutations.error} onSubmit={handleCancel} onClose={closeCancelDialog} />}
        </>}
      </>}
      {!isLoading && !error && !reservation && <Button render={<Link to="/reservas" />}>Volver a reservas</Button>}
    </section>
  )
}
