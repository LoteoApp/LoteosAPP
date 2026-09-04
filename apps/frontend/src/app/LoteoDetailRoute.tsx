import { useAuth } from '../features/auth/hooks/use-auth'
import { getUserRole, ROLE } from '../shared/auth/roles'
import LoteoDetailPage from '../features/lots/pages/LoteoDetailPage'

export default function LoteoDetailRoute() {
  const { session, user } = useAuth()
  const role = getUserRole(user ?? session?.user)
  const canEdit = role === ROLE.administrador || role === ROLE.agrimensor

  return <LoteoDetailPage accessToken={session?.access_token ?? null} canEdit={canEdit} />
}
