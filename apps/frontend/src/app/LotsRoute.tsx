import { useAuth } from '../features/auth/hooks/use-auth'
import LotsPage from '../features/lots/pages/LotsPage'

export default function LotsRoute() {
  const { session } = useAuth()

  return <LotsPage accessToken={session?.access_token ?? null} />
}
