import { useEffect, useRef, useState, type ReactNode } from 'react'
import { Link, useParams } from 'react-router'
import { Alert, AlertDescription, AlertTitle } from '../../../shared/ui/alert'
import { Button } from '../../../shared/ui/button'
import { downloadReservationReceipt, getReservation } from '../api/reservations'
import CancelReservationDialog from '../components/CancelReservationDialog'
import ReservationDetails from '../components/ReservationDetails'
import ReservationDetailsPlanSkeleton from '../components/ReservationDetailsPlanSkeleton'
import { useReservationMutations } from '../hooks/use-reservation-mutations'
import type { Reservation } from '../types'

type ReservationDetailsPageProps = {
  accessToken?: string
  renderPlan?: (reservation: Reservation) => ReactNode
}

export default function ReservationDetailsPage({ accessToken = '', renderPlan }: ReservationDetailsPageProps) {
  const token = accessToken
  const { id = '' } = useParams()
  const [reservation, setReservation] = useState<Reservation | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [receiptError, setReceiptError] = useState<string | null>(null)
  const [isDownloadingReceipt, setIsDownloadingReceipt] = useState(false)
  const [isCancelDialogOpen, setIsCancelDialogOpen] = useState(false)
  const mutations = useReservationMutations(token)
  const queryEnabled = token !== ''
  const latestRequestRef = useRef({ id, token })
  const requestKey = JSON.stringify([id, token])
  const [loadedKey, setLoadedKey] = useState(requestKey)
  if (requestKey !== loadedKey) {
    setLoadedKey(requestKey)
    setReservation(null)
    setIsLoading(queryEnabled)
    setError(null)
    setIsCancelDialogOpen(false)
  }

  useEffect(() => {
    latestRequestRef.current = { id, token }
  }, [id, token])

  useEffect(() => {
    if (!queryEnabled) return
    const controller = new AbortController()
    getReservation(token, id, controller.signal)
      .then((loaded) => { if (!controller.signal.aborted) { setReservation(loaded); setError(null) } })
      .catch((loadError: unknown) => { if (!controller.signal.aborted) setError(loadError instanceof Error ? loadError.message : 'No se pudo cargar la reserva.') })
      .finally(() => { if (!controller.signal.aborted) setIsLoading(false) })
    return () => controller.abort()
  }, [id, queryEnabled, token])

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

  async function handleDownloadReceipt() {
    if (!reservation) return
    setIsDownloadingReceipt(true)
    setReceiptError(null)
    try {
      const blob = await downloadReservationReceipt(token, reservation.id)
      const url = URL.createObjectURL(blob)
      const anchor = document.createElement('a')
      anchor.href = url
      anchor.download = `comprobante-reserva-${reservation.id}.pdf`
      anchor.click()
      window.setTimeout(() => URL.revokeObjectURL(url), 1000)
    } catch (downloadError) {
      setReceiptError(downloadError instanceof Error ? downloadError.message : 'No se pudo descargar el comprobante.')
    } finally {
      setIsDownloadingReceipt(false)
    }
  }

  function closeCancelDialog() {
    setIsCancelDialogOpen(false)
    mutations.reset()
  }

  return (
    <section className="flex flex-col gap-4">
      <Link to="/reservas" className="w-fit text-sm text-muted-foreground hover:text-foreground">Volver a reservas</Link>
      {queryEnabled && isLoading && <ReservationDetailsPlanSkeleton />}
      {queryEnabled && error && <Alert variant="destructive"><AlertTitle>No se pudo cargar la reserva</AlertTitle><AlertDescription>{error}</AlertDescription></Alert>}
      {queryEnabled && !isLoading && !error && reservation && <>
        {renderPlan?.(reservation)}
        <ReservationDetails reservation={reservation} />
        <div className="flex flex-wrap items-center gap-2">
          <Button variant="outline" className="w-fit" onClick={handleDownloadReceipt} disabled={isDownloadingReceipt}>
            {isDownloadingReceipt ? 'Generando comprobante…' : 'Descargar comprobante'}
          </Button>
          {reservation.estado === 'activa' && reservation.puedeCancelar === true && <Button variant="outline" className="w-fit" onClick={() => { mutations.reset(); setIsCancelDialogOpen(true) }}>Cancelar</Button>}
          {isCancelDialogOpen && <CancelReservationDialog reservation={reservation} isSubmitting={mutations.isSubmitting} error={mutations.error} onSubmit={handleCancel} onClose={closeCancelDialog} />}
        </div>
        {receiptError && <Alert variant="destructive"><AlertTitle>No se pudo descargar el comprobante</AlertTitle><AlertDescription>{receiptError}</AlertDescription></Alert>}
      </>}
      {queryEnabled && !isLoading && !error && !reservation && <Button render={<Link to="/reservas" />}>Volver a reservas</Button>}
    </section>
  )
}
