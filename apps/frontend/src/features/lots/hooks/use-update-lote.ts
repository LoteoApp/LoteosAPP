import { updateLote } from '../api/update-lote'
import type { LoteFormValues, UpdateLotePayload } from '../lib/loteFormValues'
import type { LoteoLote } from '../types'
import { useUpdateResource, type UpdateResourceState } from './use-update-resource'

export type UpdateLoteState = UpdateResourceState<keyof LoteFormValues>

export type UseUpdateLoteResult = UpdateLoteState & {
  update: (
    loteoId: string,
    loteId: string,
    payload: UpdateLotePayload,
  ) => Promise<LoteoLote | null>
  reset: () => void
}

function fieldForCode(code: string): keyof LoteFormValues | undefined {
  switch (code) {
    case 'lote_numero_in_use':
    case 'invalid_lote_numero':
      return 'numero'
    case 'invalid_precio':
      return 'precio'
    case 'invalid_moneda':
    case 'currency_without_price':
      return 'moneda'
    case 'invalid_superficie':
      return 'superficie'
    case 'lote_caracteristicas_too_long':
      return 'caracteristicas'
    default:
      return undefined
  }
}

export function useUpdateLote(accessToken: string | null): UseUpdateLoteResult {
  return useUpdateResource<LoteoLote, UpdateLotePayload, keyof LoteFormValues>(
    accessToken,
    updateLote,
    fieldForCode,
  )
}
