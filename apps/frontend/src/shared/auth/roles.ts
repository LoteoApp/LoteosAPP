// Roles de dominio; deben coincidir con las constantes Rol* de
// apps/backend/internal/business/domain/rol.go.
export const ROLE = {
  administrador: 'administrador',
  administrativo: 'administrativo',
  agrimensor: 'agrimensor',
  escribano: 'escribano',
  inmobiliaria: 'inmobiliaria',
} as const

export type DomainRole = (typeof ROLE)[keyof typeof ROLE]

type RoleSource = {
  app_metadata?: { role?: string | null; [key: string]: unknown } | null
}

export function getUserRole(user: RoleSource | null | undefined): string | null {
  const role = user?.app_metadata?.role
  return typeof role === 'string' && role !== '' ? role : null
}
