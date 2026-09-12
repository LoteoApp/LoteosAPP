import { Link } from 'react-router'
import { Button } from '../../../shared/ui/button'
import { Card, CardContent } from '../../../shared/ui/card'
import { formatDateTime } from '../../../shared/lib/formatDateTime'
import type { Reservation } from '../types'
import ReservationStatusBadge from './ReservationStatusBadge'

export default function ReservationsList({ reservations, onCancel }: {
  reservations: Reservation[]
  onCancel: (reservation: Reservation) => void
}) {
  if (reservations.length === 0) return <p className="text-sm text-muted-foreground">No hay reservas que coincidan con los filtros.</p>
  return (
    <ul className="grid gap-3">
      {reservations.map((reservation) => (
        <li key={reservation.id}>
          <Card>
            <CardContent className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center">
              <div className="grid gap-1">
                <div className="flex flex-wrap items-center gap-2">
                  <Link to={`/reservas/${reservation.id}`} className="font-medium hover:underline">{reservation.loteoNombre}</Link>
                  <Link to={`/lotes/${reservation.loteoId}`} className="text-sm text-muted-foreground hover:underline" aria-label={`Ver loteo ${reservation.loteoNombre}`}>Ver loteo</Link>
                  <ReservationStatusBadge state={reservation.estado} />
                </div>
                <p className="text-sm">{reservation.loteNumero ? `Lote ${reservation.loteNumero}` : 'Lote sin número'} · {reservation.cliente.nombre} {reservation.cliente.apellido}</p>
                <p className="text-sm text-muted-foreground">Vence: {formatDateTime(reservation.fechaVencimiento)}</p>
              </div>
              <div className="flex flex-wrap gap-2 sm:justify-end">
                <Button render={(
                  <Link
                    to={`/reservas/${reservation.id}`}
                    aria-label={`Ver detalle de la reserva del ${reservation.loteNumero ? `lote ${reservation.loteNumero}` : 'lote sin número'} en ${reservation.loteoNombre}`}
                  />
                )}>
                  Ver detalle
                </Button>
                {reservation.estado === 'activa' && reservation.puedeCancelar === true && <Button variant="outline" onClick={() => onCancel(reservation)}>Cancelar</Button>}
              </div>
            </CardContent>
          </Card>
        </li>
      ))}
    </ul>
  )
}
