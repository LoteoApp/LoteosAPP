import { Badge } from '../../../shared/ui/badge'
import { LOT_STATE_BADGE_STYLES, LOT_STATE_LABELS } from '../lib/lotStateVisuals'
import type { LotState } from '../types'

type LotStateBadgeProps = {
  state: LotState
}

export default function LotStateBadge({ state }: LotStateBadgeProps) {
  return (
    <Badge variant="outline" className={LOT_STATE_BADGE_STYLES[state]}>
      {LOT_STATE_LABELS[state]}
    </Badge>
  )
}
