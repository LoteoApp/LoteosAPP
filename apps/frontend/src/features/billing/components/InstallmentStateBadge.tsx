import { Badge } from '../../../shared/ui/badge'
import { INSTALLMENT_STATE_LABELS, type InstallmentState } from '../types'

const VARIANTS: Record<InstallmentState, 'default' | 'destructive' | 'outline'> = {
  pagada: 'default',
  vencida: 'destructive',
  pendiente: 'outline',
}

export default function InstallmentStateBadge({ state }: { state: InstallmentState }) {
  return (
    <Badge variant={VARIANTS[state]} className="print:border-black print:bg-white print:text-black">
      {INSTALLMENT_STATE_LABELS[state]}
    </Badge>
  )
}
