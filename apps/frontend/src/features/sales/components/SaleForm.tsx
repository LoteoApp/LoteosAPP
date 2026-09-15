import { useCallback, useEffect, useMemo, useState } from 'react'
import { Alert, AlertDescription } from '../../../shared/ui/alert'
import { Button } from '../../../shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '../../../shared/ui/card'
import { messageFromError } from '../../../shared/api/client'
import { useSaleSellers } from '../hooks/use-sale-sellers'
import AgencyCombobox from './AgencyCombobox'
import ClientCombobox from './ClientCombobox'
import NewClientDialog from './NewClientDialog'
import PaymentConditions from './PaymentConditions'
import SellerCombobox from './SellerCombobox'
import {
  EMPTY_PAYMENT_PLAN,
  agencyOfSeller,
  agencyOptionsFromSellers,
  buildSaleReceipt,
  defaultSeller,
  sellersOfAgency,
} from '../types'
import type {
  AgencyOption,
  ClienteOption,
  CreateSaleValues,
  LoteOption,
  NewClientValues,
  PaymentMethod,
  PaymentPlanValues,
  SellerOption,
} from '../types'

export type SaleFormProps = {
  // The lote being sold. Who may sell it depends on its loteo, so the
  // sellers are (re)loaded whenever it changes.
  lote: LoteOption | null
  loadClientes: (signal?: AbortSignal) => Promise<ClienteOption[]>
  createCliente: (values: NewClientValues) => Promise<ClienteOption>
  loadSellers: (loteoId: string, signal?: AbortSignal) => Promise<SellerOption[]>
  onSubmit: (values: CreateSaleValues) => Promise<boolean>
  isSubmitting?: boolean
  error?: string | null
  disabled?: boolean
}

export default function SaleForm({
  lote,
  loadClientes,
  createCliente,
  loadSellers,
  onSubmit,
  isSubmitting = false,
  error = null,
  disabled = false,
}: SaleFormProps) {
  const [clientes, setClientes] = useState<ClienteOption[]>([])
  const [isLoadingClientes, setIsLoadingClientes] = useState(true)
  const [clientesError, setClientesError] = useState<string | null>(null)

  const [paymentMethod, setPaymentMethod] = useState<PaymentMethod>('contado')
  const [paymentPlan, setPaymentPlan] = useState<PaymentPlanValues>(EMPTY_PAYMENT_PLAN)
  const [cliente, setCliente] = useState<ClienteOption | null>(null)
  const [agency, setAgency] = useState<AgencyOption | null>(null)
  const [seller, setSeller] = useState<SellerOption | null>(null)

  const [isDialogOpen, setIsDialogOpen] = useState(false)
  const [isCreatingCliente, setIsCreatingCliente] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)

  useEffect(() => {
    const controller = new AbortController()

    async function load() {
      setIsLoadingClientes(true)
      try {
        const loaded = await loadClientes(controller.signal)
        if (controller.signal.aborted) {
          return
        }
        setClientes(loaded)
        setClientesError(null)
      } catch (loadError) {
        if (controller.signal.aborted) {
          return
        }
        setClientesError(messageFromError(loadError))
      } finally {
        if (!controller.signal.aborted) {
          setIsLoadingClientes(false)
        }
      }
    }

    void load()

    return () => controller.abort()
  }, [loadClientes])

  const loteoId = lote === null ? '' : lote.loteoId
  const sellersState = useSaleSellers(loteoId, loadSellers)

  // A different loteo invalidates both selectors before the new list arrives.
  const [sellersLoteoId, setSellersLoteoId] = useState(loteoId)
  if (sellersLoteoId !== loteoId) {
    setSellersLoteoId(loteoId)
    setAgency(null)
    setSeller(null)
  }

  // Whoever registers the sale is the seller unless they say otherwise, so
  // the form opens already resolved once the catalog is known.
  const [preselectedFrom, setPreselectedFrom] = useState<SellerOption[]>(sellersState.sellers)
  if (preselectedFrom !== sellersState.sellers) {
    setPreselectedFrom(sellersState.sellers)
    const preselected = defaultSeller(sellersState.sellers)
    if (preselected !== null) {
      setSeller(preselected)
      setAgency(agencyOfSeller(preselected))
    }
  }

  const handleCreateCliente = useCallback(
    async (values: NewClientValues) => {
      setIsCreatingCliente(true)
      setCreateError(null)
      try {
        const created = await createCliente(values)
        setClientes((current) => [created, ...current])
        setCliente(created)
        setIsDialogOpen(false)
      } catch (creationError) {
        setCreateError(messageFromError(creationError))
      } finally {
        setIsCreatingCliente(false)
      }
    },
    [createCliente],
  )

  const agencies = useMemo(() => agencyOptionsFromSellers(sellersState.sellers), [sellersState.sellers])
  const agencySellers = useMemo(
    () => sellersOfAgency(sellersState.sellers, agency),
    [sellersState.sellers, agency],
  )

  const draft = buildSaleReceipt(
    { lote, cliente, seller, method: paymentMethod, plan: paymentPlan },
    new Date().toISOString(),
  )
  const pendingReason = draft.ok ? null : draft.error
  const isBusy = disabled || isSubmitting

  async function handleConfirm() {
    if (lote === null || cliente === null || seller === null || !draft.ok) {
      return
    }
    await onSubmit({
      loteoId: lote.loteoId,
      loteId: lote.id,
      clienteId: cliente.id,
      vendedorId: seller.id,
      modalidadPago: paymentMethod,
      ...(draft.plan === undefined ? {} : { planPago: draft.plan }),
    })
  }

  return (
    <>
      {clientesError !== null && (
        <p role="alert" className="text-sm text-destructive">
          {clientesError}
        </p>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Datos de la venta</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-6">
          <ClientCombobox
            clientes={clientes}
            value={cliente}
            onChange={setCliente}
            onRegisterClient={() => {
              setCreateError(null)
              setIsDialogOpen(true)
            }}
            isLoading={isLoadingClientes}
            disabled={isBusy}
          />
          <AgencyCombobox
            agencies={agencies}
            value={agency}
            onChange={(next) => {
              setAgency(next)
              setSeller(null)
            }}
            isLoading={sellersState.isLoading}
            disabled={isBusy || lote === null}
          />
          <SellerCombobox
            sellers={agencySellers}
            value={seller}
            onChange={setSeller}
            isLoading={sellersState.isLoading}
            disabled={isBusy || agency === null}
          />
          {sellersState.error !== null && (
            <p role="alert" className="text-sm text-destructive">
              {sellersState.error}
            </p>
          )}
          <PaymentConditions
            method={paymentMethod}
            plan={paymentPlan}
            lote={lote}
            onMethodChange={setPaymentMethod}
            onPlanChange={setPaymentPlan}
            disabled={isBusy}
          />
        </CardContent>
      </Card>

      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-end">
        {pendingReason !== null && (
          <p className="text-sm text-muted-foreground sm:mr-auto">{pendingReason}</p>
        )}
        <Button
          type="button"
          className="min-h-11 w-full sm:min-h-9 sm:w-auto"
          onClick={() => void handleConfirm()}
          disabled={isBusy || !draft.ok}
        >
          {isSubmitting ? 'Registrando venta…' : 'Confirmar venta'}
        </Button>
      </div>

      <NewClientDialog
        open={isDialogOpen}
        isSubmitting={isCreatingCliente}
        error={createError}
        onSubmit={handleCreateCliente}
        onClose={() => setIsDialogOpen(false)}
      />
    </>
  )
}
