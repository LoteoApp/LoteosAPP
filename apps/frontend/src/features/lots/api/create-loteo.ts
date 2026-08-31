import { apiFetch } from '../../../shared/api/client'
import type { CreateLoteoPayload } from '../lib/buildCreateLoteoPayload'

// CreatedLoteo is the subset of POST /api/v1/loteos the alta flow needs: the
// id is what the DXF upload is keyed on.
export type CreatedLoteo = {
  id: string
  nombre: string
}

export function createLoteo(payload: CreateLoteoPayload, token: string): Promise<CreatedLoteo> {
  return apiFetch<CreatedLoteo>('/api/v1/loteos', { method: 'POST', body: payload, token })
}
