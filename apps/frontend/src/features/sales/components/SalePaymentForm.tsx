import { useState, type ReactNode } from 'react'
import { Alert, AlertDescription } from '../../../shared/ui/alert'
import { Button } from '../../../shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '../../../shared/ui/card'
import PaymentConditions from './PaymentConditions'
import { EMPTY_PAYMENT_PLAN, buildSaleReceipt } from '../types'
import type {
  ClientOption,
  LotOption,
  PaymentMethod,
  PaymentPlanValues,
  SalePaymentTerms,
  SellerOption,
} from '../types'

export type SalePaymentFormProps = {
  lot: LotOption | null
  client: ClientOption | null
  seller: SellerOption | null
  // How the participants are chosen or shown; it renders above the terms.
  participants: ReactNode
  onSubmit: (terms: SalePaymentTerms) => Promise<boolean>
  isSubmitting?: boolean
  error?: ReactNode
  disabled?: boolean
  // Keeps the typed terms while a retry of an uncertain request is pending,
  // so the retry can't send other terms under the same attempt.
  termsLocked?: boolean
}

export default function SalePaymentForm({
  lot,
  client,
  seller,
  participants,
  onSubmit,
  isSubmitting = false,
  error = null,
  disabled = false,
  termsLocked = false,
}: SalePaymentFormProps) {
  const [paymentMethod, setPaymentMethod] = useState<PaymentMethod>('contado')
  const [paymentPlan, setPaymentPlan] = useState<PaymentPlanValues>(EMPTY_PAYMENT_PLAN)

  const draft = buildSaleReceipt(
    { lot, client, seller, method: paymentMethod, plan: paymentPlan },
    new Date().toISOString(),
  )
  const pendingReason = draft.ok ? null : draft.error
  const isBusy = disabled || isSubmitting

  async function handleConfirm() {
    if (!draft.ok || isBusy) {
      return
    }
    await onSubmit({
      modalidadPago: paymentMethod,
      ...(draft.plan === undefined ? {} : { planPago: draft.plan }),
    })
  }

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>Datos de la venta</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-6">
          {participants}
          <PaymentConditions
            method={paymentMethod}
            plan={paymentPlan}
            lot={lot}
            onMethodChange={setPaymentMethod}
            onPlanChange={setPaymentPlan}
            disabled={isBusy || termsLocked}
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
    </>
  )
}
