type DisplayNameSource = {
  email?: string | null
  user_metadata?: { full_name?: string | null } | null
}

export function resolveDisplayName(
  user: DisplayNameSource | null | undefined,
  fallback = '',
): string {
  return (
    [user?.user_metadata?.full_name, user?.email].find(
      (value): value is string => typeof value === 'string' && value !== '',
    ) ?? fallback
  )
}
