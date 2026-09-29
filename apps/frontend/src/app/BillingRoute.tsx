import { useAuth } from '../features/auth/hooks/use-auth'
import BillingPage from '../features/billing/pages/BillingPage'

export default function BillingRoute() {
  const { session } = useAuth()
  return <BillingPage accessToken={session?.access_token ?? ''} />
}
