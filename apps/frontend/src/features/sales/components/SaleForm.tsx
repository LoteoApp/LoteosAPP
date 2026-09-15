import { useMemo, useState, type ReactNode } from 'react'
import { Alert, AlertDescription } from '../../../shared/ui/alert'
import { Button } from '../../../shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '../../../shared/ui/card'
import { useSaleSellers } from '../hooks/use-sale-sellers'
import AgencyCombobox from './AgencyCombobox'
import ClientCombobox from './ClientCombobox'
import PaymentConditions from './PaymentConditions'
import SellerCombobox from './SellerCombobox'
import {
  DIRECT_SALE,
  EMPTY_PAYMENT_PLAN,
  agencyOfSeller,
  agencyOptionsFromSellers,
  buildSaleReceipt,
  defaultSeller,
  sellersOfAgency,
} from '../types'
import type {
  AgencyOption,
  ClientOption,
  CreateSaleValues,
  LotOption,
  PaymentMethod,
  PaymentPlanValues,
  SellerOption,
} from '../types'

export type SaleFormProps = {
  // The lote being sold. Who may sell it depends on its loteo, so the
  // sellers are (re)loaded whenever it changes.
  lot: LotOption | null
  clients: readonly ClientOption[]
  clientsLoading?: boolean
  clientsError?: string | null
  // A cliente registered from the dialog `app` renders through
  // renderClientDialog; the form selects it as soon as it arrives.
  createdClient?: ClientOption | null
  onRegisterClient?: () => void
  renderClientDialog?: ReactNode
  loadSellers: (developmentId: string, signal?: AbortSignal) => Promise<SellerOption[]>
  onSubmit: (values: CreateSaleValues) => Promise<boolean>
  isSubmitting?: boolean
  error?: string | null
  disabled?: boolean
}

export default function SaleForm({
  lot,
  clients,
  clientsLoading = false,
  clientsError = null,
  createdClient = null,
  onRegisterClient,
  renderClientDialog,
  loadSellers,
  onSubmit,
  isSubmitting = false,
  error = null,
  disabled = false,
}: SaleFormProps) {
  const [paymentMethod, setPaymentMethod] = useState<PaymentMethod>('contado')
  const [paymentPlan, setPaymentPlan] = useState<PaymentPlanValues>(EMPTY_PAYMENT_PLAN)
  const [client, setClient] = useState<ClientOption | null>(null)
  const [agency, setAgency] = useState<AgencyOption | null>(null)
  const [seller, setSeller] = useState<SellerOption | null>(null)

  const [seenCreatedClient, setSeenCreatedClient] = useState(createdClient)
  if (seenCreatedClient !== createdClient) {
    setSeenCreatedClient(createdClient)
    if (createdClient !== null) {
      setClient(createdClient)
    }
  }

  const developmentId = lot === null ? '' : lot.developmentId
  const sellersState = useSaleSellers(developmentId, loadSellers)

  // A different loteo invalidates both selectors before the new list arrives.
  const [sellersDevelopmentId, setSellersDevelopmentId] = useState(developmentId)
  if (sellersDevelopmentId !== developmentId) {
    setSellersDevelopmentId(developmentId)
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

  const agencies = useMemo(() => agencyOptionsFromSellers(sellersState.sellers), [sellersState.sellers])
  // An internal user selling directly sells at their own name: the seller is
  // theirs to keep, not to choose. Picking an agency reopens the choice.
  const actorSeller = useMemo(
    () => sellersState.sellers.find((candidate) => candidate.esActor === true) ?? null,
    [sellersState.sellers],
  )
  const directSaleSeller =
    actorSeller !== null && agencyOfSeller(actorSeller).id === DIRECT_SALE.id ? actorSeller : null
  const sellerIsFixedToActor = directSaleSeller !== null && agency?.id === DIRECT_SALE.id
  const agencySellers = useMemo(
    () => sellersOfAgency(sellersState.sellers, agency),
    [sellersState.sellers, agency],
  )

  const draft = buildSaleReceipt(
    { lot, client, seller, method: paymentMethod, plan: paymentPlan },
    new Date().toISOString(),
  )
  const pendingReason = draft.ok ? null : draft.error
  const isBusy = disabled || isSubmitting

  async function handleConfirm() {
    if (lot === null || client === null || seller === null || !draft.ok) {
      return
    }
    await onSubmit({
      loteoId: lot.developmentId,
      loteId: lot.id,
      clienteId: client.id,
      vendedorId: seller.id,
      modalidadPago: paymentMethod,
      ...(draft.plan === undefined ? {} : { planPago: draft.plan }),
    })
  }

  return (
    <>
      {clientsError !== null && (
        <p role="alert" className="text-sm text-destructive">
          {clientsError}
        </p>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Datos de la venta</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-6">
          <ClientCombobox
            clients={clients}
            value={client}
            onChange={setClient}
            onRegisterClient={() => onRegisterClient?.()}
            isLoading={clientsLoading}
            disabled={isBusy}
          />
          <AgencyCombobox
            agencies={agencies}
            value={agency}
            onChange={(next) => {
              setAgency(next)
              setSeller(next?.id === DIRECT_SALE.id ? directSaleSeller : null)
            }}
            isLoading={sellersState.isLoading}
            disabled={isBusy || lot === null}
          />
          <SellerCombobox
            sellers={agencySellers}
            value={seller}
            onChange={setSeller}
            isLoading={sellersState.isLoading}
            disabled={isBusy || agency === null || sellerIsFixedToActor}
            description={
              sellerIsFixedToActor
                ? 'Vendés a tu nombre. Elegí una inmobiliaria para registrar la venta de uno de sus vendedores.'
                : undefined
            }
          />
          {sellersState.error !== null && (
            <p role="alert" className="text-sm text-destructive">
              {sellersState.error}
            </p>
          )}
          <PaymentConditions
            method={paymentMethod}
            plan={paymentPlan}
            lot={lot}
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

      {renderClientDialog}
    </>
  )
}
