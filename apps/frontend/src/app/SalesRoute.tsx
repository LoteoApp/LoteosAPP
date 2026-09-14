import { useCallback } from 'react'
import { useAuth } from '../features/auth/hooks/use-auth'
import { createClient, listClients } from '../features/clients/api/clients'
import { listEligibleSellers } from '../features/reservations/api/reservations'
import { searchLotes } from '../features/sales/api/search-lotes'
import SalesPage from '../features/sales/pages/SalesPage'
import type { NewClientValues } from '../features/sales/types'

export default function SalesRoute() {
  const { session } = useAuth()
  const token = session?.access_token ?? ''

  const loadLotes = useCallback(
    (signal?: AbortSignal) => searchLotes(token, { estado: 'disponible' }, signal),
    [token],
  )

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
    (loteoId: string, signal?: AbortSignal) =>
      listEligibleSellers(token, loteoId, signal, 'agencia'),
    [token],
  )

  return (
    <SalesPage
      loadLotes={loadLotes}
      loadClientes={loadClientes}
      createCliente={createCliente}
      loadSellers={loadSellers}
    />
  )
}
