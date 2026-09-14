import { useCallback, useMemo } from 'react'
import { useParams } from 'react-router'
import { useAuth } from '../features/auth/hooks/use-auth'
import { createClient, listClients } from '../features/clients/api/clients'
import { useLoteo } from '../features/lots/hooks/use-loteo'
import { listEligibleSellers } from '../features/reservations/api/reservations'
import SaleCreatePage from '../features/sales/pages/SaleCreatePage'
import type { NewClientValues, SaleCreateDevelopment } from '../features/sales/types'
import ReservationLoteoPlan from './ReservationLoteoPlan'

export default function SaleCreateRoute() {
  const { session } = useAuth()
  const { loteoId = '', loteId = '' } = useParams()
  const token = session?.access_token ?? ''
  const loteoState = useLoteo(loteoId, token)

  const sourceLoteo = loteoState.status === 'loaded' ? loteoState.loteo : null
  const loteo = useMemo<SaleCreateDevelopment | null>(() => sourceLoteo ? ({
    id: sourceLoteo.id,
    nombre: sourceLoteo.nombre,
    ubicacion: sourceLoteo.ubicacion,
    descripcion: sourceLoteo.descripcion,
    manzanas: sourceLoteo.manzanas,
    lotes: sourceLoteo.lotes,
  }) : null, [sourceLoteo])

  const loadClientes = useCallback(
    (signal?: AbortSignal) => listClients(token, signal),
    [token],
  )

  const createCliente = useCallback(
    (values: NewClientValues) => createClient(token, values),
    [token],
  )

  // A venta lists every agency with sellers, assigned to the loteo or not, and
  // lets an agency user pick a colleague; a reserva does neither.
  const loadSellers = useCallback(
    (targetLoteoId: string, signal?: AbortSignal) =>
      listEligibleSellers(token, targetLoteoId, signal, 'agencia'),
    [token],
  )

  return (
    <SaleCreatePage
      loteoId={loteoId}
      loteId={loteId}
      loteo={loteo}
      loteoStatus={loteoState.status}
      loteoError={loteoState.status === 'error' ? loteoState.message : undefined}
      loadClientes={loadClientes}
      createCliente={createCliente}
      loadSellers={loadSellers}
      renderPlan={sourceLoteo && <ReservationLoteoPlan loteo={sourceLoteo} selectedLoteId={loteId} variant="reference" />}
    />
  )
}
