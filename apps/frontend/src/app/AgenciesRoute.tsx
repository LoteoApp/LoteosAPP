import { useAuth } from '../features/auth/hooks/use-auth'
import AgenciesPage from '../features/agencies/pages/AgenciesPage'
import { getUserRole, ROLE } from '../shared/auth/roles'

export default function AgenciesRoute() {
  const { session, user } = useAuth()

  return (
    <AgenciesPage
      accessToken={session?.access_token ?? null}
      isAdmin={getUserRole(user) === ROLE.administrador}
    />
  )
}
