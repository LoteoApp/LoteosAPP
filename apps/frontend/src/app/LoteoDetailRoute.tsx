import { useAuth } from '../features/auth/hooks/use-auth'
import LoteoDetailPage from '../features/lots/pages/LoteoDetailPage'

export default function LoteoDetailRoute() {
  const { session } = useAuth()

  return <LoteoDetailPage accessToken={session?.access_token ?? null} />
}
