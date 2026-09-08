import { Badge } from '../../../shared/ui/badge'
import type { LotState } from '../types'

type LotStateBadgeProps = {
  state: LotState
}

const labels: Record<LotState, string> = {
  disponible: 'Disponible',
  reservado: 'Reservado',
  vendido: 'Vendido',
  finalizado: 'Finalizado',
}

const styles: Record<LotState, string> = {
  disponible:
    'border-lot-available-foreground/20 bg-lot-available text-lot-available-foreground',
  reservado:
    'border-lot-reserved-foreground/20 bg-lot-reserved text-lot-reserved-foreground',
  vendido: 'border-lot-sold-foreground/20 bg-lot-sold text-lot-sold-foreground',
  finalizado:
    'border-lot-completed-foreground/20 bg-lot-completed text-lot-completed-foreground',
}

export default function LotStateBadge({ state }: LotStateBadgeProps) {
  return (
    <Badge variant="outline" className={styles[state]}>
      {labels[state]}
    </Badge>
  )
}
