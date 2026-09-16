export type Agency = {
  id: string
  businessName: string
}

export const MOCK_AGENCIES: readonly Agency[] = [
  { id: 'inm-san-martin', businessName: 'Inmobiliaria San Martín' },
  { id: 'inm-lotes-del-sur', businessName: 'Lotes del Sur' },
  { id: 'inm-altamira', businessName: 'Altamira Propiedades' },
  { id: 'inm-rio-parana', businessName: 'Río Paraná Inmuebles' },
  { id: 'inm-campos-y-lotes', businessName: 'Campos y Lotes SRL' },
]

export function listAgencies(): readonly Agency[] {
  return MOCK_AGENCIES
}
