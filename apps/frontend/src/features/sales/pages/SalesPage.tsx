import { useCallback, useEffect, useMemo, useState } from 'react'
import { Button } from '../../../shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '../../../shared/ui/card'
import { messageFromError } from '../../../shared/api/client'
import AgencyCombobox from '../components/AgencyCombobox'
import ClientCombobox from '../components/ClientCombobox'
import LoteCombobox from '../components/LoteCombobox'
import NewClientDialog from '../components/NewClientDialog'
import PaymentConditions from '../components/PaymentConditions'
import SaleReceiptDialog from '../components/SaleReceiptDialog'
import SellerCombobox from '../components/SellerCombobox'
import {
  agencyOfSeller,
  agencyOptionsFromSellers,
  buildSaleReceipt,
  defaultSeller,
  sellersOfAgency,
} from '../types'
import type {
  AgencyOption,
  ClienteOption,
  LoteOption,
  NewClientValues,
  PaymentMethod,
  SaleReceipt,
  SellerOption,
} from '../types'

export type SalesPageProps = {
  // The composition root injects every data source: sales never imports
  // another feature's files.
  loadLotes: (signal?: AbortSignal) => Promise<LoteOption[]>
  loadClientes: (signal?: AbortSignal) => Promise<ClienteOption[]>
  createCliente: (values: NewClientValues) => Promise<ClienteOption>
  // Eligibility depends on the loteo of the lote being sold, so the sellers
  // can only be loaded once there is a lote.
  loadSellers: (loteoId: string, signal?: AbortSignal) => Promise<SellerOption[]>
}

export default function SalesPage({
  loadLotes,
  loadClientes,
  createCliente,
  loadSellers,
}: SalesPageProps) {
  const [lotes, setLotes] = useState<LoteOption[]>([])
  const [clientes, setClientes] = useState<ClienteOption[]>([])
  const [sellers, setSellers] = useState<SellerOption[]>([])

  const [lote, setLote] = useState<LoteOption | null>(null)
  const [paymentMethod, setPaymentMethod] = useState<PaymentMethod>('contado')
  const [cliente, setCliente] = useState<ClienteOption | null>(null)
  const [agency, setAgency] = useState<AgencyOption | null>(null)
  const [seller, setSeller] = useState<SellerOption | null>(null)

  const [isLoading, setIsLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [isLoadingSellers, setIsLoadingSellers] = useState(false)
  const [sellersError, setSellersError] = useState<string | null>(null)

  const [receipt, setReceipt] = useState<SaleReceipt | null>(null)

  const [isDialogOpen, setIsDialogOpen] = useState(false)
  const [isCreatingCliente, setIsCreatingCliente] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)

  useEffect(() => {
    const controller = new AbortController()

    async function load() {
      setIsLoading(true)
      try {
        const [loadedLotes, loadedClientes] = await Promise.all([
          loadLotes(controller.signal),
          loadClientes(controller.signal),
        ])
        if (controller.signal.aborted) {
          return
        }
        setLotes(loadedLotes)
        setClientes(loadedClientes)
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
  }, [loadLotes, loadClientes])

  const loteoId = lote === null ? '' : lote.loteoId
  const [sellersLoteoId, setSellersLoteoId] = useState(loteoId)

  // Who can sell depends on the loteo, so a different lote invalidates both
  // selectors before the new list arrives.
  if (sellersLoteoId !== loteoId) {
    setSellersLoteoId(loteoId)
    setSellers([])
    setAgency(null)
    setSeller(null)
    setSellersError(null)
  }

  useEffect(() => {
    if (loteoId === '') {
      return
    }

    const controller = new AbortController()

    async function load() {
      setIsLoadingSellers(true)
      try {
        const loaded = await loadSellers(loteoId, controller.signal)
        if (controller.signal.aborted) {
          return
        }
        setSellers(loaded)
        const preselected = defaultSeller(loaded)
        if (preselected !== null) {
          setSeller(preselected)
          setAgency(agencyOfSeller(preselected))
        }
      } catch (error) {
        if (controller.signal.aborted) {
          return
        }
        setSellersError(messageFromError(error))
      } finally {
        if (!controller.signal.aborted) {
          setIsLoadingSellers(false)
        }
      }
    }

    void load()

    return () => controller.abort()
  }, [loteoId, loadSellers])

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

  const agencies = useMemo(() => agencyOptionsFromSellers(sellers), [sellers])
  const agencySellers = useMemo(() => sellersOfAgency(sellers, agency), [sellers, agency])

  const sale = buildSaleReceipt(
    { lote, cliente, seller, method: paymentMethod },
    new Date().toISOString(),
  )
  const readyReceipt = sale.ok ? sale.receipt : null
  const pendingReason = sale.ok ? null : sale.error

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
          <AgencyCombobox
            agencies={agencies}
            value={agency}
            onChange={(next) => {
              setAgency(next)
              setSeller(null)
            }}
            isLoading={isLoadingSellers}
            disabled={lote === null}
          />
          <SellerCombobox
            sellers={agencySellers}
            value={seller}
            onChange={setSeller}
            isLoading={isLoadingSellers}
            disabled={agency === null}
          />
          {sellersError !== null && (
            <p role="alert" className="text-sm text-destructive">
              {sellersError}
            </p>
          )}
          <PaymentConditions
            method={paymentMethod}
            lote={lote}
            onMethodChange={setPaymentMethod}
            disabled={isLoading}
          />
        </CardContent>
      </Card>

      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-end">
        {pendingReason !== null && (
          <p className="text-sm text-muted-foreground sm:mr-auto">{pendingReason}</p>
        )}
        <Button
          type="button"
          className="min-h-11 w-full sm:min-h-9 sm:w-auto"
          onClick={() => setReceipt(readyReceipt)}
          disabled={isLoading || readyReceipt === null}
        >
          Confirmar venta
        </Button>
      </div>

      <NewClientDialog
        open={isDialogOpen}
        isSubmitting={isCreatingCliente}
        error={createError}
        onSubmit={handleCreateCliente}
        onClose={() => setIsDialogOpen(false)}
      />

      <SaleReceiptDialog
        open={receipt !== null}
        receipt={receipt}
        onClose={() => setReceipt(null)}
      />
    </section>
  )
}
