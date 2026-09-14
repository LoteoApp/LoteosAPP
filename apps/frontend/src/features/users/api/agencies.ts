import { listAgencies as listAgencyItems } from '../../../shared/api/agencies'

// The user creation form only needs enough to label a picker option; the
// full Agency shape (cuit, telefono, email) belongs to the agencies feature.
export type AgencyOption = {
  id: string
  razonSocial: string
}

export async function listAgencies(token: string, signal?: AbortSignal): Promise<AgencyOption[]> {
  const items = await listAgencyItems(token, signal)
  return items.map(({ id, razonSocial }) => ({ id, razonSocial }))
}
