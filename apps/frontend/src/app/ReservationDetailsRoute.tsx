import { useAuth } from '../features/auth/hooks/use-auth'
import ReservationDetailsPage from '../features/reservations/pages/ReservationDetailsPage'
import { ReservationDetailsPlan } from './ReservationLoteoPlan'

export default function ReservationDetailsRoute() {
  const { session } = useAuth()
  const token = session?.access_token ?? ''
  return <ReservationDetailsPage accessToken={token} renderPlan={(reservation) => <ReservationDetailsPlan accessToken={token} reservation={reservation} />} />
}
