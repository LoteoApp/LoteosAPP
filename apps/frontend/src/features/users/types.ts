export const GESTIONABLE_ROLES = ['administrativo', 'escribano', 'inmobiliaria', 'agrimensor'] as const

export type GestionableRol = (typeof GESTIONABLE_ROLES)[number]

export const ROLE_LABELS: Record<GestionableRol, string> = {
  administrativo: 'Administrativo',
  agrimensor: 'Agrimensor',
  escribano: 'Escribano',
  inmobiliaria: 'Inmobiliaria',
}

export type Usuario = {
  id: string
  email: string
  nombre: string
  apellido: string
  rol: GestionableRol
  inmobiliariaId: string | null
  perfilCompleto: boolean
  fechaBaja: string | null
  createdAt: string
}

export function isActivo(usuario: Usuario): boolean {
  return usuario.fechaBaja === null
}

// The agency an inmobiliaria user is linked to when it's created. Only the
// fields the dropdown needs; the full agency lives in the agencies feature.
export type AgencyOption = {
  id: string
  razonSocial: string
}

export type UsuarioFormValues = {
  nombre: string
  apellido: string
  email: string
  rol: GestionableRol
  inmobiliariaId?: string
}

export type UsuarioUpdateValues = {
  nombre: string
  apellido: string
}

export function toUsuarioUpdateValues(usuario: Usuario): UsuarioUpdateValues {
  const { nombre, apellido } = usuario
  return { nombre, apellido }
}
