import { useAuth } from '../features/auth/hooks/use-auth'
import DebtStatementPage from '../features/billing/pages/DebtStatementPage'

export default function DebtStatementRoute() {
  const { session } = useAuth()
  return <DebtStatementPage accessToken={session?.access_token ?? ''} />
}
