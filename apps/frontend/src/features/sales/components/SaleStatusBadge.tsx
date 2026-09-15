import { Badge } from '../../../shared/ui/badge'
import { SALE_STATE_LABELS, type SaleState } from '../types'

export default function SaleStatusBadge({ state }: { state: SaleState }) {
  return <Badge variant={state === 'cancelada' ? 'outline' : 'default'}>{SALE_STATE_LABELS[state]}</Badge>
}
