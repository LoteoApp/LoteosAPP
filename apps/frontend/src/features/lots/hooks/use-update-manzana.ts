import { updateManzana, type UpdateManzanaPayload } from '../api/update-manzana'
import type { LoteoManzana } from '../types'
import { useUpdateResource, type UpdateResourceState } from './use-update-resource'

export type ManzanaFormValues = {
  numero: string
  tieneAgua: boolean
  tieneCloaca: boolean
  tieneLuz: boolean
  tieneGas: boolean
  calleIds: string[]
}

export type UpdateManzanaState = UpdateResourceState<'numero' | 'calleIds'>

export type UseUpdateManzanaResult = UpdateManzanaState & {
  update: (
    loteoId: string,
    manzanaId: string,
    payload: UpdateManzanaPayload,
  ) => Promise<LoteoManzana | null>
  reset: () => void
}

const FIELD_BY_CODE: Record<string, 'numero' | 'calleIds'> = {
  manzana_numero_in_use: 'numero',
  invalid_manzana_numero: 'numero',
  too_many_manzana_calles: 'calleIds',
  unknown_calle: 'calleIds',
  duplicate_manzana_calle: 'calleIds',
}

function fieldForCode(code: string): 'numero' | 'calleIds' | undefined {
  return FIELD_BY_CODE[code]
}

export function useUpdateManzana(accessToken: string | null): UseUpdateManzanaResult {
  return useUpdateResource<LoteoManzana, UpdateManzanaPayload, 'numero' | 'calleIds'>(
    accessToken,
    updateManzana,
    fieldForCode,
  )
}
