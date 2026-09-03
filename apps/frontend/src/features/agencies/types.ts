// The property names are the wire contract of /api/v1/inmobiliarias, so they
// keep the Spanish the API publishes even though the symbols are in English.
export type Agency = {
  id: string
  razonSocial: string
  cuit: string
  telefono: string
  email: string
}

export type AgencyFormValues = Omit<Agency, 'id'>

export function toAgencyFormValues(agency: Agency): AgencyFormValues {
  const { razonSocial, cuit, telefono, email } = agency
  return { razonSocial, cuit, telefono, email }
}
