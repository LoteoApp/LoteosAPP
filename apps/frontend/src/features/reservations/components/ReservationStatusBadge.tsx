import { Badge } from '../../../shared/ui/badge'
import type { ReservationState } from '../types'

const labels: Record<ReservationState, string> = {
  activa: 'Activa',
  vencida: 'Vencida',
  cancelada: 'Cancelada',
  convertida: 'Convertida',
}

export default function ReservationStatusBadge({ state }: { state: ReservationState }) {
  return <Badge variant={state === 'activa' ? 'default' : 'outline'}>{labels[state]}</Badge>
}
