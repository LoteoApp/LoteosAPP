import { useCallback, useEffect, useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '../../../shared/ui/card'
import { messageFromError } from '../../../shared/api/client'
import AgencyCombobox from '../components/AgencyCombobox'
import ClientCombobox from '../components/ClientCombobox'
import LoteCombobox from '../components/LoteCombobox'
import NewClientDialog from '../components/NewClientDialog'
import PaymentConditions from '../components/PaymentConditions'
import type {
  AgencyOption,
  ClienteOption,
  LoteOption,
  NewClientValues,
  PaymentMethod,
} from '../types'

export type SalesPageProps = {
  // The composition root injects every data source: sales never imports
  // another feature's files.
  loadLotes: (signal?: AbortSignal) => Promise<LoteOption[]>
  loadClientes: (signal?: AbortSignal) => Promise<ClienteOption[]>
  createCliente: (values: NewClientValues) => Promise<ClienteOption>
  // Absent when the caller is an inmobiliaria: the sale belongs to its own
  // agency, so there is nothing to pick.
  loadAgencies?: (signal?: AbortSignal) => Promise<AgencyOption[]>
}

export default function SalesPage({
  loadLotes,
  loadClientes,
  createCliente,
  loadAgencies,
}: SalesPageProps) {
  const [lotes, setLotes] = useState<LoteOption[]>([])
  const [clientes, setClientes] = useState<ClienteOption[]>([])
  const [agencies, setAgencies] = useState<AgencyOption[]>([])

  const [lote, setLote] = useState<LoteOption | null>(null)
  const [paymentMethod, setPaymentMethod] = useState<PaymentMethod>('contado')
  const [cliente, setCliente] = useState<ClienteOption | null>(null)
  const [agency, setAgency] = useState<AgencyOption | null>(null)

  const [isLoading, setIsLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)

  const [isDialogOpen, setIsDialogOpen] = useState(false)
  const [isCreatingCliente, setIsCreatingCliente] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)

  useEffect(() => {
    const controller = new AbortController()

    async function load() {
      setIsLoading(true)
      try {
        const [loadedLotes, loadedClientes, loadedAgencies] = await Promise.all([
          loadLotes(controller.signal),
          loadClientes(controller.signal),
          loadAgencies ? loadAgencies(controller.signal) : Promise.resolve([]),
        ])
        if (controller.signal.aborted) {
          return
        }
        setLotes(loadedLotes)
        setClientes(loadedClientes)
        setAgencies(loadedAgencies)
        setLoadError(null)
      } catch (error) {
        if (controller.signal.aborted) {
          return
        }
        setLoadError(messageFromError(error))
      } finally {
        if (!controller.signal.aborted) {
          setIsLoading(false)
        }
      }
    }

    void load()

    return () => controller.abort()
  }, [loadLotes, loadClientes, loadAgencies])

  const handleCreateCliente = useCallback(
    async (values: NewClientValues) => {
      setIsCreatingCliente(true)
      setCreateError(null)
      try {
        const created = await createCliente(values)
        setClientes((current) => [created, ...current])
        setCliente(created)
        setIsDialogOpen(false)
      } catch (error) {
        setCreateError(messageFromError(error))
      } finally {
        setIsCreatingCliente(false)
      }
    },
    [createCliente],
  )

  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-col gap-2">
        <h1 className="text-2xl font-semibold text-foreground">Nueva venta</h1>
        <p className="text-muted-foreground">
          Elegí el lote, el cliente comprador y quién realizó la venta.
        </p>
      </div>

      {loadError !== null && (
        <p role="alert" className="text-sm text-destructive">
          {loadError}
        </p>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Datos de la venta</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-6">
          <LoteCombobox
            lotes={lotes}
            value={lote}
            onChange={setLote}
            isLoading={isLoading}
          />
          <ClientCombobox
            clientes={clientes}
            value={cliente}
            onChange={setCliente}
            onRegisterClient={() => {
              setCreateError(null)
              setIsDialogOpen(true)
            }}
            isLoading={isLoading}
          />
          {loadAgencies && (
            <AgencyCombobox
              agencies={agencies}
              value={agency}
              onChange={setAgency}
              isLoading={isLoading}
            />
          )}
          <PaymentConditions
            method={paymentMethod}
            lote={lote}
            onMethodChange={setPaymentMethod}
            disabled={isLoading}
          />
        </CardContent>
      </Card>

      <NewClientDialog
        open={isDialogOpen}
        isSubmitting={isCreatingCliente}
        error={createError}
        onSubmit={handleCreateCliente}
        onClose={() => setIsDialogOpen(false)}
      />
    </section>
  )
}
