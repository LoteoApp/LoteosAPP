import { useCallback, useMemo, useState } from 'react'
import { useParams } from 'react-router'
import { useAuth } from '../features/auth/hooks/use-auth'
import CreateClientDialog from '../features/clients/components/CreateClientDialog'
import { useClients } from '../features/clients/hooks/use-clients'
import type { Cliente } from '../features/clients/types'
import { useLoteo } from '../features/lots/hooks/use-loteo'
import { listEligibleSellers } from '../features/reservations/api/reservations'
import { createSale } from '../features/sales/api/sales'
import SaleCreatePage from '../features/sales/pages/SaleCreatePage'
import type { CreateSaleValues, SaleCreateDevelopment } from '../features/sales/types'
import ReservationLoteoPlan from './ReservationLoteoPlan'

export default function SaleCreateRoute() {
  const { session } = useAuth()
  const { loteoId = '', loteId = '' } = useParams()
  const token = session?.access_token ?? ''
  const developmentState = useLoteo(loteoId, token)
  const clientsState = useClients(token, { enabled: token !== '' })
  const [createdClient, setCreatedClient] = useState<Cliente | null>(null)
  const [isClientDialogOpen, setIsClientDialogOpen] = useState(false)

  const sourceDevelopment = developmentState.status === 'loaded' ? developmentState.loteo : null
  const development = useMemo<SaleCreateDevelopment | null>(() => sourceDevelopment ? ({
    id: sourceDevelopment.id,
    nombre: sourceDevelopment.nombre,
    ubicacion: sourceDevelopment.ubicacion,
    descripcion: sourceDevelopment.descripcion,
    manzanas: sourceDevelopment.manzanas,
    lotes: sourceDevelopment.lotes,
  }) : null, [sourceDevelopment])
  const clients = useMemo<Cliente[]>(() => [...(createdClient ? [createdClient] : []), ...clientsState.clientes]
    .filter((client, index, values) => values.findIndex((item) => item.id === client.id) === index),
  [clientsState.clientes, createdClient])

  // A venta lists every agency with sellers, assigned to the loteo or not, and
  // lets an agency user pick a colleague; a reserva does neither.
  const loadSellers = useCallback(
    (developmentId: string, signal?: AbortSignal) =>
      listEligibleSellers(token, developmentId, signal, 'agencia'),
    [token],
  )

  const registerSale = useCallback(
    (values: CreateSaleValues, idempotencyKey: string) => createSale(token, values, idempotencyKey),
    [token],
  )

  return (
    <SaleCreatePage
      loteoId={loteoId}
      loteId={loteId}
      development={development}
      developmentStatus={developmentState.status}
      developmentError={developmentState.status === 'error' ? developmentState.message : undefined}
      clients={clients}
      clientsLoading={clientsState.isLoading}
      clientsError={clientsState.error}
      createdClient={createdClient}
      onRegisterClient={() => setIsClientDialogOpen(true)}
      renderClientDialog={(
        <CreateClientDialog
          accessToken={token}
          clients={clients}
          open={isClientDialogOpen}
          onOpenChange={setIsClientDialogOpen}
          onCreated={setCreatedClient}
          description="Registrá los datos para asociarlo a la venta."
        />
      )}
      loadSellers={loadSellers}
      createSale={registerSale}
      renderPlan={sourceDevelopment && <ReservationLoteoPlan loteo={sourceDevelopment} selectedLoteId={loteId} variant="reference" />}
    />
  )
}
