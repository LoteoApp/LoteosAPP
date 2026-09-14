import { LOT_STATE_LABELS, LOT_STATE_PAINT } from '../lib/lotStateVisuals'
import { LOT_STATES, type LotState } from '../types'

type LotStateLegendProps = {
  states?: readonly LotState[]
  counts?: Readonly<Record<LotState, number>>
}

export default function LotStateLegend({ states = LOT_STATES, counts }: LotStateLegendProps) {
  if (states.length === 0) {
    return null
  }

  return (
    <ul
      aria-label="Colores por estado del lote"
      className="flex flex-wrap items-center gap-x-3 gap-y-1"
    >
      {states.map((state) => (
        <li key={state} className="flex items-center gap-1.5 text-xs text-muted-foreground">
          <span
            aria-hidden
            className="size-2.5 shrink-0 rounded-full"
            style={{ backgroundColor: LOT_STATE_PAINT[state].fill }}
          />
          {LOT_STATE_LABELS[state]}
          {counts && (
            <span className="font-semibold tabular-nums text-foreground">{counts[state]}</span>
          )}
        </li>
      ))}
    </ul>
  )
}
