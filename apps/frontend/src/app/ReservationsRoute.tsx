import { useAuth } from '../features/auth/hooks/use-auth'
import ReservationsPage from '../features/reservations/pages/ReservationsPage'

export default function ReservationsRoute() {
  const { session } = useAuth()
  return <ReservationsPage accessToken={session?.access_token ?? ''} />
}
