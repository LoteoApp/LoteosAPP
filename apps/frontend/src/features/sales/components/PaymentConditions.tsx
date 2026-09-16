import { Field, FieldDescription, FieldLabel } from '../../../shared/ui/field'
import { Input } from '../../../shared/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectList,
  SelectTrigger,
  SelectValue,
} from '../../../shared/ui/select'
import { formatCurrency } from '../../../shared/lib/formatCurrency'
import { installmentsLabel } from './installmentsLabel'
import {
  PAYMENT_METHODS,
  PAYMENT_METHOD_LABELS,
  PAYMENT_PERIODS,
  PAYMENT_PERIOD_LABELS,
  buildPaymentSchedule,
  lastInstallmentAmount,
  isFinancedMethod,
  saleDisabledReason,
  parsePaymentPlan,
  type LotOption,
  type PaymentMethod,
  type PaymentPeriod,
  type PaymentPlanValues,
} from '../types'

type PaymentConditionsProps = {
  method: PaymentMethod
  plan: PaymentPlanValues
  // The amount of a sale is the price of the lote, so it is shown, never
  // typed; a financed plan splits it.
  lot: LotOption | null
  onMethodChange: (method: PaymentMethod) => void
  onPlanChange: (plan: PaymentPlanValues) => void
  disabled?: boolean
}

const METHOD_DESCRIPTIONS: Record<PaymentMethod, string> = {
  contado: 'El monto es el precio del lote seleccionado.',
  financiado: 'El precio del lote se divide en cuotas, con el interés que indiques.',
  entrega_financiada: 'Una entrega inicial y el resto del precio en cuotas.',
}

function AmountValue({ lot }: { lot: LotOption | null }) {
  if (lot === null) {
    return <p className="text-sm text-muted-foreground">Elegí un lote para ver el monto.</p>
  }
  const disabledReason = saleDisabledReason(lot)
  if (disabledReason !== null || lot.price === null) {
    return (
      <p role="alert" className="text-sm text-destructive">
        {disabledReason}
      </p>
    )
  }

  return (
    <p className="text-lg font-semibold tabular-nums text-foreground">
      {formatCurrency(lot.price, lot.currency)}
    </p>
  )
}

function PlanPreview({
  method,
  plan,
  lot,
}: {
  method: PaymentMethod
  plan: PaymentPlanValues
  lot: LotOption
}) {
  if (lot.price === null || saleDisabledReason(lot) !== null) {
    return null
  }
  const parsed = parsePaymentPlan(method, plan, lot.price)
  if (!parsed.ok || parsed.plan === undefined) {
    return (
      <p className="text-sm text-muted-foreground">
        Completá el plan para ver el detalle de las cuotas.
      </p>
    )
  }
  const schedule = buildPaymentSchedule(lot.price, parsed.plan)
  const installmentsText = installmentsLabel(
    schedule.installments.length,
    schedule.installmentAmount,
    lastInstallmentAmount(schedule),
    lot.currency,
  )

  return (
    <dl
      aria-label="Detalle del plan"
      className="grid grid-cols-1 gap-x-6 gap-y-3 rounded-lg border border-border bg-muted/40 p-4 text-sm sm:grid-cols-2"
    >
      {method === 'entrega_financiada' && (
        <PreviewItem term="Entrega" value={formatCurrency(parsed.plan.montoEntrega, lot.currency)} />
      )}
      <PreviewItem term="Monto financiado" value={formatCurrency(schedule.financedAmount, lot.currency)} />
      <PreviewItem term="Cuotas" value={installmentsText} />
      <PreviewItem term="Total financiado" value={formatCurrency(schedule.totalAmount, lot.currency)} />
    </dl>
  )
}

function PreviewItem({ term, value }: { term: string; value: string }) {
  return (
    <div className="flex flex-col gap-0.5">
      <dt className="text-xs font-medium text-muted-foreground">{term}</dt>
      <dd className="font-medium tabular-nums text-foreground">{value}</dd>
    </div>
  )
}

export default function PaymentConditions({
  method,
  plan,
  lot,
  onMethodChange,
  onPlanChange,
  disabled = false,
}: PaymentConditionsProps) {
  const financed = isFinancedMethod(method)
  const update = <Key extends keyof PaymentPlanValues>(key: Key, value: PaymentPlanValues[Key]) =>
    onPlanChange({ ...plan, [key]: value })

  return (
    <div className="flex flex-col gap-6">
      <Field>
        <FieldLabel htmlFor="venta-modalidad">Condiciones de pago</FieldLabel>
        <Select
          name="modalidad"
          value={method}
          onValueChange={(next) => onMethodChange(next as PaymentMethod)}
          disabled={disabled}
        >
          <SelectTrigger id="venta-modalidad" className="min-h-11 md:min-h-9">
            <SelectValue>{(current: PaymentMethod) => PAYMENT_METHOD_LABELS[current]}</SelectValue>
          </SelectTrigger>
          <SelectContent>
            <SelectList>
              {PAYMENT_METHODS.map((candidate) => (
                <SelectItem key={candidate} value={candidate} className="min-h-11 md:min-h-8">
                  {PAYMENT_METHOD_LABELS[candidate]}
                </SelectItem>
              ))}
            </SelectList>
          </SelectContent>
        </Select>
        <FieldDescription>{METHOD_DESCRIPTIONS[method]}</FieldDescription>
      </Field>

      <div className="flex flex-col gap-1.5">
        <span className="text-sm leading-none font-medium">
          {financed ? 'Precio del lote' : 'Monto'}
        </span>
        <AmountValue lot={lot} />
      </div>

      {financed && (
        <div className="flex flex-col gap-4">
          {method === 'entrega_financiada' && (
            <Field>
              <FieldLabel htmlFor="venta-entrega">Monto de entrega</FieldLabel>
              <Input
                id="venta-entrega"
                name="downPayment"
                inputMode="decimal"
                autoComplete="off"
                value={plan.downPayment}
                onChange={(event) => update('downPayment', event.target.value)}
                disabled={disabled}
                className="min-h-11 md:min-h-9"
              />
              <FieldDescription>
                {lot === null ? 'En la moneda del lote.' : `En ${lot.currency || 'la moneda del lote'}.`}
              </FieldDescription>
            </Field>
          )}

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <Field>
              <FieldLabel htmlFor="venta-cuotas">Cantidad de cuotas</FieldLabel>
              <Input
                id="venta-cuotas"
                name="installments"
                inputMode="numeric"
                autoComplete="off"
                value={plan.installments}
                onChange={(event) => update('installments', event.target.value)}
                disabled={disabled}
                className="min-h-11 md:min-h-9"
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="venta-tasa">Tasa de interés (%)</FieldLabel>
              <Input
                id="venta-tasa"
                name="interestRate"
                inputMode="decimal"
                autoComplete="off"
                placeholder="0"
                value={plan.interestRate}
                onChange={(event) => update('interestRate', event.target.value)}
                disabled={disabled}
                className="min-h-11 md:min-h-9"
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="venta-periodicidad">Periodicidad</FieldLabel>
              <Select
                name="period"
                value={plan.period}
                onValueChange={(next) => update('period', next as PaymentPeriod)}
                disabled={disabled}
              >
                <SelectTrigger id="venta-periodicidad" className="min-h-11 md:min-h-9">
                  <SelectValue>
                    {(current: PaymentPeriod) => PAYMENT_PERIOD_LABELS[current]}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  <SelectList>
                    {PAYMENT_PERIODS.map((candidate) => (
                      <SelectItem key={candidate} value={candidate} className="min-h-11 md:min-h-8">
                        {PAYMENT_PERIOD_LABELS[candidate]}
                      </SelectItem>
                    ))}
                  </SelectList>
                </SelectContent>
              </Select>
            </Field>
          </div>
          <FieldDescription>
            La tasa se aplica una sola vez sobre el monto financiado. La primera cuota vence un
            período después de la venta.
          </FieldDescription>

          {lot !== null && <PlanPreview method={method} plan={plan} lot={lot} />}
        </div>
      )}
    </div>
  )
}
