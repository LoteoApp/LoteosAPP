import { Field, FieldDescription, FieldLabel } from '../../../shared/ui/field'
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
  isPaymentMethodAvailable,
  type LoteOption,
  type PaymentMethod,
} from '../types'

type PaymentConditionsProps = {
  method: PaymentMethod
  // The amount of a contado sale is the price of the lote, so it is shown,
  // never typed.
  lote: LoteOption | null
  onMethodChange: (method: PaymentMethod) => void
  disabled?: boolean
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

export default function PaymentConditions({
  method,
  lote,
  onMethodChange,
  disabled = false,
}: PaymentConditionsProps) {
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
              {PAYMENT_METHODS.map((candidate) => {
                const available = isPaymentMethodAvailable(candidate)
                return (
                  <SelectItem
                    key={candidate}
                    value={candidate}
                    disabled={!available}
                    className="min-h-11 md:min-h-8"
                  >
                    {available
                      ? PAYMENT_METHOD_LABELS[candidate]
                      : `${PAYMENT_METHOD_LABELS[candidate]} (próximamente)`}
                  </SelectItem>
                )
              })}
            </SelectList>
          </SelectContent>
        </Select>
        <FieldDescription>
          Por ahora solo se puede registrar una venta al contado.
        </FieldDescription>
      </Field>

      {method === 'contado' && (
        <div className="flex flex-col gap-1.5">
          <span className="text-sm leading-none font-medium">Monto</span>
          <AmountValue lote={lote} />
          <FieldDescription>Es el precio del lote seleccionado.</FieldDescription>
        </div>
      )}
    </div>
  )
}
