import { useCallback, useMemo } from 'react'
import { useAuth } from '../features/auth/hooks/use-auth'
import { getUserRole, ROLE } from '../shared/auth/roles'
import { listAgencies } from '../features/agencies/api/agencies'
import { createClient, listClients } from '../features/clients/api/clients'
import { searchLotes } from '../features/sales/api/search-lotes'
import SalesPage from '../features/sales/pages/SalesPage'
import type { NewClientValues } from '../features/sales/types'

export default function SalesRoute() {
  const { session, user } = useAuth()
  const token = session?.access_token ?? ''
  // An inmobiliaria sells for its own agency, so it never picks one.
  const isAgencyUser = getUserRole(user) === ROLE.inmobiliaria

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

  const loadAgencies = useMemo(
    () =>
      isAgencyUser ? undefined : (signal?: AbortSignal) => listAgencies(token, signal),
    [isAgencyUser, token],
  )

  return (
    <SalesPage
      loadLotes={loadLotes}
      loadClientes={loadClientes}
      createCliente={createCliente}
      loadAgencies={loadAgencies}
    />
  )
}
