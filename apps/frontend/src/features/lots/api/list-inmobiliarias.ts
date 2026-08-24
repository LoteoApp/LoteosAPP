export type Inmobiliaria = {
  id: string
  razonSocial: string
}

export const MOCK_INMOBILIARIAS: readonly Inmobiliaria[] = [
  { id: 'inm-san-martin', razonSocial: 'Inmobiliaria San Martín' },
  { id: 'inm-lotes-del-sur', razonSocial: 'Lotes del Sur' },
  { id: 'inm-altamira', razonSocial: 'Altamira Propiedades' },
  { id: 'inm-rio-parana', razonSocial: 'Río Paraná Inmuebles' },
  { id: 'inm-campos-y-lotes', razonSocial: 'Campos y Lotes SRL' },
]

export function listInmobiliarias(): readonly Inmobiliaria[] {
  return MOCK_INMOBILIARIAS
}
