import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card'
import { formatDateTime } from '../../../shared/lib/formatDateTime'
import type { Reservation } from '../types'
import ReservationStatusBadge from './ReservationStatusBadge'

export default function ReservationDetails({ reservation }: { reservation: Reservation }) {
  return (
    <div className="grid gap-4">
      <Card>
        <CardHeader>
          <div className="flex flex-wrap items-center gap-2"><CardTitle>{reservation.loteoNombre}</CardTitle><ReservationStatusBadge state={reservation.estado} /></div>
          <CardDescription>{reservation.loteNumero ? `Lote ${reservation.loteNumero}` : 'Lote sin número'}</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3 text-sm sm:grid-cols-2">
          <Info label="Cliente" value={`${reservation.cliente.nombre} ${reservation.cliente.apellido} · DNI ${reservation.cliente.dni}`} />
          <Info label="Vendedor" value={`${reservation.vendedor.nombre} ${reservation.vendedor.apellido}`} />
          <Info label="Cargada por" value={`${reservation.usuarioAlta.nombre} ${reservation.usuarioAlta.apellido}`} />
          <Info label="Creada" value={formatDateTime(reservation.fechaCreacion)} />
          <Info label="Vencimiento" value={formatDateTime(reservation.fechaVencimiento)} />
        </CardContent>
      </Card>
      <Card>
        <CardHeader><CardTitle>Historial</CardTitle></CardHeader>
        <CardContent>
          <ol className="grid gap-3">
            {(reservation.historial ?? []).map((entry) => (
              <li key={entry.id} className="border-l-2 border-border pl-3 text-sm">
                <div className="flex flex-wrap gap-2"><ReservationStatusBadge state={entry.estado} /><span className="text-muted-foreground">{formatDateTime(entry.fecha)}</span></div>
                {entry.razon && <p className="mt-1 text-muted-foreground">{entry.razon}</p>}
                {entry.usuario && <p className="mt-1 text-muted-foreground">Por {entry.usuario.nombre} {entry.usuario.apellido}</p>}
              </li>
            ))}
          </ol>
        </CardContent>
      </Card>
    </div>
  )
}

function Info({ label, value }: { label: string; value: string }) {
  return <div><dt className="text-xs font-medium text-muted-foreground">{label}</dt><dd>{value}</dd></div>
}
