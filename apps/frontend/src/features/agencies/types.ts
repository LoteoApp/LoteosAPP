export type Inmobiliaria = {
  id: string
  razonSocial: string
  cuit: string
  telefono: string
  email: string
}

export type InmobiliariaFormValues = Omit<Inmobiliaria, 'id'>

export function toInmobiliariaFormValues(
  inmobiliaria: Inmobiliaria,
): InmobiliariaFormValues {
  const { razonSocial, cuit, telefono, email } = inmobiliaria
  return { razonSocial, cuit, telefono, email }
}
