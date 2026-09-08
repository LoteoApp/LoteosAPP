import { useState } from 'react'
import { Link } from 'react-router'
import { Alert, AlertDescription } from '../../../shared/ui/alert'
import { Button } from '../../../shared/ui/button'
import CancelReservationDialog from '../components/CancelReservationDialog'
import ReservationFilters from '../components/ReservationFilters'
import ReservationsList from '../components/ReservationsList'
import { useReservationMutations } from '../hooks/use-reservation-mutations'
import { useReservations } from '../hooks/use-reservations'
import type { ReservationState } from '../types'

type ReservationsPageProps = { accessToken?: string }

export default function ReservationsPage({ accessToken = '' }: ReservationsPageProps) {
  const token = accessToken
  const [search, setSearch] = useState('')
  const [state, setState] = useState<ReservationState | ''>('')
  const [pageNumber, setPageNumber] = useState(1)
  const [cancelingId, setCancelingId] = useState<string | null>(null)
  const mutations = useReservationMutations(token)
  const reservations = useReservations(token, { q: search || undefined, estado: state || undefined, pagina: pageNumber })

  const selectedReservation = reservations.page.reservas.find((item) => item.id === cancelingId) ?? null

  async function handleCancel(reason: string) {
    if (!selectedReservation) return false
    const canceled = await mutations.cancel(selectedReservation.id, reason)
    if (!canceled) return false
    setCancelingId(null)
    reservations.refresh()
    return true
  }

  function closeCancelDialog() {
    setCancelingId(null)
    mutations.reset()
  }

  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-foreground">Reservas</h1>
          <p className="text-sm text-muted-foreground">Consultá y gestioná las reservas creadas desde el visor.</p>
        </div>
        <Button render={<Link to="/lotes" />}>Abrir visor de lotes</Button>
      </div>
      <section className="grid gap-3">
        <div>
          <h2 className="text-lg font-semibold">Reservas cargadas</h2>
          <p className="text-sm text-muted-foreground">Para crear una reserva, abrí un loteo y seleccioná un lote disponible en el visor.</p>
        </div>
        <ReservationFilters
          search={search}
          state={state}
          onSearchChange={(value) => { setSearch(value); setPageNumber(1) }}
          onStateChange={(value) => { setState(value); setPageNumber(1) }}
        />
        {reservations.error && <Alert variant="destructive"><AlertDescription>{reservations.error}</AlertDescription></Alert>}
        {reservations.isLoading ? <p className="text-sm text-muted-foreground">Cargando reservas…</p> : <ReservationsList reservations={reservations.page.reservas} onCancel={(reservation) => { mutations.reset(); setCancelingId(reservation.id) }} />}
        {reservations.page.paginas > 1 && (
          <nav className="flex flex-wrap items-center justify-between gap-3" aria-label="Paginación de reservas">
            <p className="text-sm text-muted-foreground">
              Página {reservations.page.pagina} de {reservations.page.paginas} · {reservations.page.total} reservas
            </p>
            <div className="flex gap-2">
              <Button type="button" variant="outline" disabled={reservations.page.pagina <= 1 || reservations.isLoading} onClick={() => setPageNumber((page) => Math.max(1, page - 1))}>
                Anterior
              </Button>
              <Button type="button" variant="outline" disabled={reservations.page.pagina >= reservations.page.paginas || reservations.isLoading} onClick={() => setPageNumber((page) => Math.min(reservations.page.paginas, page + 1))}>
                Siguiente
              </Button>
            </div>
          </nav>
        )}
      </section>
      {selectedReservation && <CancelReservationDialog reservation={selectedReservation} isSubmitting={mutations.isSubmitting} error={mutations.error} onSubmit={handleCancel} onClose={closeCancelDialog} />}
    </section>
  )
}
