import { useAuth } from '../features/auth/hooks/use-auth'
import UsersPage from '../features/users/pages/UsersPage'

export default function UsersRoute() {
  const { session } = useAuth()

  return <UsersPage accessToken={session?.access_token ?? null} />
}
