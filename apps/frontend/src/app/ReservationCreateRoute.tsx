import { useMemo, useState } from 'react'
import { UserPlus } from 'lucide-react'
import { useParams } from 'react-router'
import { getUserRole, ROLE } from '../shared/auth/roles'
import { Button } from '../shared/ui/button'
import { useAuth } from '../features/auth/hooks/use-auth'
import CreateClientDialog from '../features/clients/components/CreateClientDialog'
import { useClients } from '../features/clients/hooks/use-clients'
import type { Cliente } from '../features/clients/types'
import { useLoteo } from '../features/lots/hooks/use-loteo'
import type { LoteoDetail } from '../features/lots/types'
import ReservationCreatePage from '../features/reservations/pages/ReservationCreatePage'
import type { ReservationCreateDevelopment } from '../features/reservations/types'
import ReservationLoteoPlan from './ReservationLoteoPlan'

export default function ReservationCreateRoute() {
  const { session, user } = useAuth()
  const { loteoId = '', loteId = '' } = useParams()
  const token = session?.access_token ?? ''
  const isAgencyUser = getUserRole(user ?? session?.user) === ROLE.inmobiliaria
  const loteoState = useLoteo(loteoId, token)
  const clientsState = useClients(token, { enabled: token !== '' })
  const [createdClient, setCreatedClient] = useState<Cliente | null>(null)
  const [isClientDialogOpen, setIsClientDialogOpen] = useState(false)

  const sourceLoteo = loteoState.status === 'loaded' ? loteoState.loteo : null
  const loteo = useMemo<ReservationCreateDevelopment | null>(() => sourceLoteo ? ({
    id: sourceLoteo.id,
    nombre: sourceLoteo.nombre,
    ubicacion: sourceLoteo.ubicacion,
    descripcion: sourceLoteo.descripcion,
    manzanas: sourceLoteo.manzanas,
    lotes: sourceLoteo.lotes,
  }) : null, [sourceLoteo])
  const clients = useMemo<Cliente[]>(() => [...clientsState.clientes, ...(createdClient ? [createdClient] : [])]
    .filter((client, index, values) => values.findIndex((item) => item.id === client.id) === index),
  [clientsState.clientes, createdClient])

  return (
    <ReservationCreatePage
      accessToken={token}
      loteoId={loteoId}
      loteId={loteId}
      loteo={loteo}
      loteoStatus={loteoState.status}
      loteoError={loteoState.status === 'error' ? loteoState.message : undefined}
      clients={clients}
      clientsLoading={clientsState.isLoading}
      clientsError={clientsState.error}
      isAgencyUser={isAgencyUser}
      renderPlan={sourceLoteo && <ReservationLoteoPlan loteo={sourceLoteo} selectedLoteId={loteId} variant="reference" />}
      renderClientAction={<Button type="button" variant="outline" size="sm" onClick={() => setIsClientDialogOpen(true)}><UserPlus aria-hidden />Nuevo cliente</Button>}
      renderClientDialog={(
        <CreateClientDialog
          accessToken={token}
          clients={clients}
          open={isClientDialogOpen}
          onOpenChange={setIsClientDialogOpen}
          onCreated={setCreatedClient}
        />
      )}
    />
  )
}
