import { useAuth } from '../features/auth/hooks/use-auth'
import SalesPage from '../features/sales/pages/SalesPage'

export default function SalesRoute() {
  const { session } = useAuth()
  return <SalesPage accessToken={session?.access_token ?? ''} />
}
