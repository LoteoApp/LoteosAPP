import { cn } from '../../../shared/lib/utils'
import { formatArea } from '../../../shared/lib/formatArea'
import { formatCurrency } from '../../../shared/lib/formatCurrency'
import { LOT_STATE_LABELS, LOT_STATE_PAINT } from '../lib/lotStateVisuals'
import type { LoteoLote } from '../types'
import LotStateBadge from './LotStateBadge'

type LotesListProps = {
  lotes: LoteoLote[]
  manzanaNumberById: ReadonlyMap<string, string>
  selectedLoteId?: string | null
  onSelectLote?: (loteId: string) => void
}

const EMPTY_VALUE = '—'

function rowLabel(lote: LoteoLote, manzana: string | undefined): string {
  return [
    lote.numero ? `Lote ${lote.numero}` : 'Lote',
    manzana ? `Mz ${manzana}` : null,
    LOT_STATE_LABELS[lote.estado],
  ]
    .filter((part): part is string => part !== null)
    .join(' · ')
}

export default function LotesList({
  lotes,
  manzanaNumberById,
  selectedLoteId = null,
  onSelectLote,
}: LotesListProps) {
  if (lotes.length === 0) {
    return (
      <p className="px-1 py-6 text-center text-sm text-muted-foreground">
        No hay lotes que coincidan con la búsqueda.
      </p>
    )
  }

  return (
    <ul aria-label="Lotes del loteo" className="flex flex-col">
      {lotes.map((lote) => {
        const selected = lote.id === selectedLoteId
        const manzana = manzanaNumberById.get(lote.manzanaId)
        return (
          <li key={lote.id}>
            <button
              type="button"
              aria-label={rowLabel(lote, manzana)}
              aria-pressed={onSelectLote ? selected : undefined}
              disabled={!onSelectLote}
              onClick={() => onSelectLote?.(lote.id)}
              className={cn(
                'flex w-full min-h-11 flex-wrap items-center gap-x-2 gap-y-1 rounded-md px-2 py-2 text-left text-sm',
                onSelectLote && 'cursor-pointer hover:bg-muted/60',
                selected && 'bg-muted ring-1 ring-border',
              )}
            >
              <span
                aria-hidden
                className="size-2.5 shrink-0 rounded-full"
                style={{ backgroundColor: LOT_STATE_PAINT[lote.estado].fill }}
              />
              <span className="font-medium">{lote.numero ? `Lote ${lote.numero}` : 'Lote'}</span>
              <span className="text-muted-foreground">
                {manzana ? `· Mz ${manzana}` : ''}
              </span>
              <span className="ml-auto tabular-nums text-muted-foreground">
                {lote.superficie === null ? EMPTY_VALUE : formatArea(lote.superficie)}
              </span>
              <span className="w-28 text-right tabular-nums">
                {lote.precio === null ? EMPTY_VALUE : formatCurrency(lote.precio, lote.moneda)}
              </span>
              <span aria-hidden>
                <LotStateBadge state={lote.estado} />
              </span>
            </button>
          </li>
        )
      })}
    </ul>
  )
}
