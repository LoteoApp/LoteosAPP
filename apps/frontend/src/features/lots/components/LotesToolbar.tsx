import { Search } from 'lucide-react'
import { Input } from '../../../shared/ui/input'
import { ToggleGroup, ToggleGroupItem } from '../../../shared/ui/toggle-group'
import { LOT_STATE_LABELS, LOT_STATE_PAINT } from '../lib/lotStateVisuals'
import { LOT_STATES, isLotState, type LotState } from '../types'

type LotesToolbarProps = {
  search: string
  onSearchChange: (search: string) => void
  states: ReadonlySet<LotState>
  onStatesChange: (states: ReadonlySet<LotState>) => void
  counts: Readonly<Record<LotState, number>>
}

export default function LotesToolbar({
  search,
  onSearchChange,
  states,
  onStatesChange,
  counts,
}: LotesToolbarProps) {
  return (
    <div className="flex flex-col gap-2 md:flex-row md:items-center">
      <div className="relative flex-1">
        <Search
          aria-hidden
          className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
        />
        <Input
          type="search"
          value={search}
          aria-label="Buscar lote o manzana"
          placeholder="Buscar lote o manzana"
          className="pl-9"
          onChange={(event) => onSearchChange(event.target.value)}
        />
      </div>

      <ToggleGroup
        multiple
        variant="outline"
        size="sm"
        aria-label="Filtrar por estado"
        value={[...states]}
        onValueChange={(next) => onStatesChange(new Set(next.filter(isLotState)))}
      >
        {LOT_STATES.map((state) => (
          <ToggleGroupItem
            key={state}
            value={state}
            aria-label={LOT_STATE_LABELS[state]}
            className="min-h-11 min-w-11 gap-1.5 px-2 text-xs md:min-h-8"
          >
            <span
              aria-hidden
              className="size-2.5 shrink-0 rounded-full"
              style={{ backgroundColor: LOT_STATE_PAINT[state].fill }}
            />
            <span className="tabular-nums">{counts[state]}</span>
          </ToggleGroupItem>
        ))}
      </ToggleGroup>
    </div>
  )
}
