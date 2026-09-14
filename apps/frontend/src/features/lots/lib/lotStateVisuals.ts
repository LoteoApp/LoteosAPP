import type { LoteoLote, LotState } from '../types'

export const LOT_STATE_LABELS: Record<LotState, string> = {
  disponible: 'Disponible',
  reservado: 'Reservado',
  vendido: 'Vendido',
  finalizado: 'Finalizado',
}

export const LOT_STATE_BADGE_STYLES: Record<LotState, string> = {
  disponible:
    'border-lot-available-foreground/20 bg-lot-available text-lot-available-foreground',
  reservado:
    'border-lot-reserved-foreground/20 bg-lot-reserved text-lot-reserved-foreground',
  vendido: 'border-lot-sold-foreground/20 bg-lot-sold text-lot-sold-foreground',
  finalizado:
    'border-lot-completed-foreground/20 bg-lot-completed text-lot-completed-foreground',
}

export type LotStatePaint = {
  fill: string
  stroke: string
}

export const LOT_STATE_PAINT: Record<LotState, LotStatePaint> = {
  disponible: {
    fill: 'var(--lot-available-plan)',
    stroke: 'var(--lot-available-foreground)',
  },
  reservado: {
    fill: 'var(--lot-reserved-plan)',
    stroke: 'var(--lot-reserved-foreground)',
  },
  vendido: {
    fill: 'var(--lot-sold-plan)',
    stroke: 'var(--lot-sold-foreground)',
  },
  finalizado: {
    fill: 'var(--lot-completed-plan)',
    stroke: 'var(--lot-completed-foreground)',
  },
}

export function countLotesByState(lotes: readonly LoteoLote[]): Record<LotState, number> {
  const counts: Record<LotState, number> = {
    disponible: 0,
    reservado: 0,
    vendido: 0,
    finalizado: 0,
  }
  for (const lote of lotes) {
    counts[lote.estado] += 1
  }
  return counts
}
