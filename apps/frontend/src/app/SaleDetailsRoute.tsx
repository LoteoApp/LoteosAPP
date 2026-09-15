import { useAuth } from '../features/auth/hooks/use-auth'
import SaleDetailsPage from '../features/sales/pages/SaleDetailsPage'
import { LoteReferencePlan } from './ReservationLoteoPlan'

export default function SaleDetailsRoute() {
  const { session } = useAuth()
  const token = session?.access_token ?? ''
  return (
    <SaleDetailsPage
      accessToken={token}
      renderPlan={(sale) => <LoteReferencePlan accessToken={token} loteoId={sale.loteoId} loteId={sale.loteId} />}
    />
  )
}
