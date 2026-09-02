export const GESTIONABLE_ROLES = ['administrativo', 'escribano', 'inmobiliaria'] as const

export type GestionableRol = (typeof GESTIONABLE_ROLES)[number]

export const ROLE_LABELS: Record<GestionableRol, string> = {
  administrativo: 'Administrativo',
  escribano: 'Escribano',
  inmobiliaria: 'Inmobiliaria',
}

export type Usuario = {
  id: string
  email: string
  nombre: string
  apellido: string
  rol: GestionableRol
  perfilCompleto: boolean
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
}

export type UsuarioUpdateValues = {
  nombre: string
  apellido: string
}

export function toUsuarioUpdateValues(usuario: Usuario): UsuarioUpdateValues {
  const { nombre, apellido } = usuario
  return { nombre, apellido }
}
