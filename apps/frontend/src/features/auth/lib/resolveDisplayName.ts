import type { UserProfile } from 'oidc-client-ts'

export function resolveDisplayName(
  profile: Pick<UserProfile, 'preferred_username' | 'name' | 'email'> | undefined,
  fallback = '',
): string {
  return (
    [profile?.preferred_username, profile?.name, profile?.email].find(
      (value) => !!value,
    ) ?? fallback
  )
}
