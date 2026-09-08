import { useAuth } from '../features/auth/hooks/use-auth'
import ReservationDetailsPage from '../features/reservations/pages/ReservationDetailsPage'

export default function ReservationDetailsRoute() {
  const { session } = useAuth()
  return <ReservationDetailsPage accessToken={session?.access_token ?? ''} />
}
