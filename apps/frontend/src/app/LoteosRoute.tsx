import { useAuth } from '../features/auth/hooks/use-auth'
import LoteosListPage from '../features/lots/pages/LoteosListPage'

export default function LoteosRoute() {
  const { session } = useAuth()

  return <LoteosListPage accessToken={session?.access_token ?? null} />
}
