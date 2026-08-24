type RoleSource = {
  app_metadata?: { role?: string | null; [key: string]: unknown } | null
}

export function getUserRole(user: RoleSource | null | undefined): string | null {
  const role = user?.app_metadata?.role
  return typeof role === 'string' && role !== '' ? role : null
}
