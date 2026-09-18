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
  // Only set for rol inmobiliaria: the agency this user operates on behalf of.
  inmobiliariaId?: string
  perfilCompleto: boolean
  // Only present in the list (and right after creating a user): the other
  // endpoints do not ask the identity provider. Absent means unknown.
  invitacionAceptada?: boolean
  fechaBaja: string | null
  createdAt: string
}

export function isActivo(usuario: Usuario): boolean {
  return usuario.fechaBaja === null
}

export type UsuarioFormValues = {
  nombre: string
  apellido: string
  email: string
  rol: GestionableRol
  // Required by the backend when rol is inmobiliaria, omitted otherwise.
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
