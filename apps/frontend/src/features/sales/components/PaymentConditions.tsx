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
import {
  PAYMENT_METHODS,
  PAYMENT_METHOD_LABELS,
  PAYMENT_PERIODS,
  PAYMENT_PERIOD_LABELS,
  buildPaymentSchedule,
  isFinancedMethod,
  parsePaymentPlan,
  type LoteOption,
  type PaymentMethod,
  type PaymentPeriod,
  type PaymentPlanValues,
} from '../types'

type PaymentConditionsProps = {
  method: PaymentMethod
  plan: PaymentPlanValues
  // The amount of a sale is the price of the lote, so it is shown, never
  // typed; a financed plan splits it.
  lote: LoteOption | null
  onMethodChange: (method: PaymentMethod) => void
  onPlanChange: (plan: PaymentPlanValues) => void
  disabled?: boolean
}

const METHOD_DESCRIPTIONS: Record<PaymentMethod, string> = {
  contado: 'El monto es el precio del lote seleccionado.',
  financiado: 'El precio del lote se divide en cuotas, con el interés que indiques.',
  entrega_financiada: 'Una entrega inicial y el resto del precio en cuotas.',
}

function AmountValue({ lote }: { lote: LoteOption | null }) {
  if (lote === null) {
    return <p className="text-sm text-muted-foreground">Elegí un lote para ver el monto.</p>
  }
  if (lote.precio === null) {
    return (
      <p role="alert" className="text-sm text-destructive">
        El lote no tiene precio cargado. Cargalo en el detalle del loteo antes de vender.
      </p>
    )
  }

  return (
    <p className="text-lg font-semibold tabular-nums text-foreground">
      {formatCurrency(lote.precio, lote.moneda)}
    </p>
  )
}

function PlanPreview({
  method,
  plan,
  lote,
}: {
  method: PaymentMethod
  plan: PaymentPlanValues
  lote: LoteOption
}) {
  if (lote.precio === null) {
    return null
  }
  const parsed = parsePaymentPlan(method, plan, lote.precio)
  if (!parsed.ok || parsed.plan === undefined) {
    return (
      <p className="text-sm text-muted-foreground">
        Completá el plan para ver el detalle de las cuotas.
      </p>
    )
  }
  const schedule = buildPaymentSchedule(lote.precio, parsed.plan)
  const lastAmount = schedule.cuotas[schedule.cuotas.length - 1]
  const cuotaLabel =
    lastAmount === schedule.montoCuota
      ? `${schedule.cuotas.length} × ${formatCurrency(schedule.montoCuota, lote.moneda)}`
      : `${schedule.cuotas.length - 1} × ${formatCurrency(schedule.montoCuota, lote.moneda)} + 1 × ${formatCurrency(lastAmount, lote.moneda)}`

  return (
    <dl
      aria-label="Detalle del plan"
      className="grid grid-cols-1 gap-x-6 gap-y-3 rounded-lg border border-border bg-muted/40 p-4 text-sm sm:grid-cols-2"
    >
      {method === 'entrega_financiada' && (
        <PreviewItem term="Entrega" value={formatCurrency(parsed.plan.montoEntrega, lote.moneda)} />
      )}
      <PreviewItem term="Monto financiado" value={formatCurrency(schedule.montoFinanciado, lote.moneda)} />
      <PreviewItem term="Cuotas" value={cuotaLabel} />
      <PreviewItem term="Total financiado" value={formatCurrency(schedule.montoTotal, lote.moneda)} />
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
  lote,
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
        <AmountValue lote={lote} />
      </div>

      {financed && (
        <div className="flex flex-col gap-4">
          {method === 'entrega_financiada' && (
            <Field>
              <FieldLabel htmlFor="venta-entrega">Monto de entrega</FieldLabel>
              <Input
                id="venta-entrega"
                name="montoEntrega"
                inputMode="decimal"
                autoComplete="off"
                value={plan.montoEntrega}
                onChange={(event) => update('montoEntrega', event.target.value)}
                disabled={disabled}
                className="min-h-11 md:min-h-9"
              />
              <FieldDescription>
                {lote === null ? 'En la moneda del lote.' : `En ${lote.moneda || 'la moneda del lote'}.`}
              </FieldDescription>
            </Field>
          )}

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <Field>
              <FieldLabel htmlFor="venta-cuotas">Cantidad de cuotas</FieldLabel>
              <Input
                id="venta-cuotas"
                name="cantidadCuotas"
                inputMode="numeric"
                autoComplete="off"
                value={plan.cantidadCuotas}
                onChange={(event) => update('cantidadCuotas', event.target.value)}
                disabled={disabled}
                className="min-h-11 md:min-h-9"
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="venta-tasa">Tasa de interés (%)</FieldLabel>
              <Input
                id="venta-tasa"
                name="tasaInteres"
                inputMode="decimal"
                autoComplete="off"
                placeholder="0"
                value={plan.tasaInteres}
                onChange={(event) => update('tasaInteres', event.target.value)}
                disabled={disabled}
                className="min-h-11 md:min-h-9"
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="venta-periodicidad">Periodicidad</FieldLabel>
              <Select
                name="periodicidad"
                value={plan.periodicidad}
                onValueChange={(next) => update('periodicidad', next as PaymentPeriod)}
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

          {lote !== null && <PlanPreview method={method} plan={plan} lote={lote} />}
        </div>
      )}
    </div>
  )
}
