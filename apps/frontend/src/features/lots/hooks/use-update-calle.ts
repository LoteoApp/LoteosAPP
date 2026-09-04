import { updateCalle, type UpdateCallePayload } from '../api/update-calle'
import type { LoteoCalle } from '../types'
import { useUpdateResource, type UpdateResourceState } from './use-update-resource'

export type UpdateCalleState = UpdateResourceState<'nombre' | 'tipo'>

export type UseUpdateCalleResult = UpdateCalleState & {
  update: (
    loteoId: string,
    calleId: string,
    payload: UpdateCallePayload,
  ) => Promise<LoteoCalle | null>
  reset: () => void
}

const FIELD_BY_CODE: Record<string, 'nombre' | 'tipo'> = {
  invalid_calle_nombre: 'nombre',
  invalid_calle_tipo: 'tipo',
}

function fieldForCode(code: string): 'nombre' | 'tipo' | undefined {
  return FIELD_BY_CODE[code]
}

export function useUpdateCalle(accessToken: string | null): UseUpdateCalleResult {
  return useUpdateResource<LoteoCalle, UpdateCallePayload, 'nombre' | 'tipo'>(
    accessToken,
    updateCalle,
    fieldForCode,
  )
}
