import { useState, type FormEvent } from 'react'
import { Alert, AlertDescription } from '../../../shared/ui/alert'
import { Button } from '../../../shared/ui/button'
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogTitle } from '../../../shared/ui/dialog'
import { Field, FieldDescription, FieldLabel } from '../../../shared/ui/field'
import { Input } from '../../../shared/ui/input'
import { Select, SelectContent, SelectItem, SelectList, SelectTrigger, SelectValue } from '../../../shared/ui/select'
import { Textarea } from '../../../shared/ui/textarea'
import { formatCurrency } from '../../../shared/lib/formatCurrency'
import {
  EMPTY_PAYMENT_TERMS,
  MAX_OBSERVATION_LENGTH,
  PAYMENT_MEDIUMS,
  PAYMENT_MEDIUM_LABELS,
  todayInputValue,
  validatePaymentTerms,
  type PaymentMedium,
  type PaymentTerms,
} from '../types'

type PaymentDialogProps = {
  open: boolean
  mode: 'pago' | 'cancelacion_total'
  amount: number
  currency: string
  // What the cobro covers, as the confirmation reads it ("Entrega + Cuota 1").
  itemsLabel: string
  isSubmitting: boolean
  error: string | null
  onSubmit: (terms: PaymentTerms) => void
  onClose: () => void
}

const COPY = {
  pago: {
    title: 'Registrar cobro',
    description: 'Confirmá el medio de pago y la fecha en que el cliente pagó.',
    submit: 'Confirmar cobro',
  },
  cancelacion_total: {
    title: 'Cancelar el saldo total',
    description:
      'Se cobra todo lo adeudado en un solo pago. La venta queda completada y el lote pasa a finalizado.',
    submit: 'Confirmar cancelación total',
  },
} as const

export default function PaymentDialog({
  open,
  mode,
  amount,
  currency,
  itemsLabel,
  isSubmitting,
  error,
  onSubmit,
  onClose,
}: PaymentDialogProps) {
  const [terms, setTerms] = useState<PaymentTerms>({ ...EMPTY_PAYMENT_TERMS, fechaPago: todayInputValue() })
  const [validationError, setValidationError] = useState<string | null>(null)
  const copy = COPY[mode]

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const problem = validatePaymentTerms(terms)
    setValidationError(problem)
    if (problem === null) {
      onSubmit(terms)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next && !isSubmitting) {
          onClose()
        }
      }}
    >
      <DialogContent className="p-4 sm:p-6">
        <DialogTitle>{copy.title}</DialogTitle>
        <DialogDescription className="mt-1">{copy.description}</DialogDescription>
        <form className="mt-4 flex flex-col gap-4" onSubmit={handleSubmit} noValidate>
          <div className="rounded-lg border border-border bg-muted/40 px-4 py-3">
            <p className="text-xs font-medium tracking-[0.12em] text-muted-foreground uppercase">Total a cobrar</p>
            <p className="text-2xl font-semibold tabular-nums">{formatCurrency(amount, currency)}</p>
            <p className="text-sm text-muted-foreground">{itemsLabel}</p>
          </div>
          <Field>
            <FieldLabel htmlFor="medio-pago">Medio de pago</FieldLabel>
            <Select
              value={terms.medioPago}
              onValueChange={(value) => setTerms({ ...terms, medioPago: value as PaymentMedium })}
            >
              <SelectTrigger id="medio-pago">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectList>
                  {PAYMENT_MEDIUMS.map((medium) => (
                    <SelectItem key={medium} value={medium}>
                      {PAYMENT_MEDIUM_LABELS[medium]}
                    </SelectItem>
                  ))}
                </SelectList>
              </SelectContent>
            </Select>
          </Field>
          <Field>
            <FieldLabel htmlFor="fecha-pago">Fecha de pago</FieldLabel>
            <Input
              id="fecha-pago"
              type="date"
              max={todayInputValue()}
              value={terms.fechaPago}
              onChange={(event) => setTerms({ ...terms, fechaPago: event.target.value })}
            />
            <FieldDescription>Dejala vacía para registrar el cobro con la fecha de hoy.</FieldDescription>
          </Field>
          <Field>
            <FieldLabel htmlFor="observacion-pago">Observación</FieldLabel>
            <Textarea
              id="observacion-pago"
              placeholder="Número de comprobante, banco, etc."
              maxLength={MAX_OBSERVATION_LENGTH}
              value={terms.observacion}
              onChange={(event) => setTerms({ ...terms, observacion: event.target.value })}
            />
          </Field>
          {(validationError ?? error) !== null && (
            <Alert variant="destructive">
              <AlertDescription>{validationError ?? error}</AlertDescription>
            </Alert>
          )}
          <div className="flex flex-col gap-2 sm:flex-row sm:justify-end">
            <DialogClose
              render={
                <Button type="button" variant="outline" className="min-h-11 sm:min-h-9" disabled={isSubmitting}>
                  Cancelar
                </Button>
              }
            />
            <Button type="submit" className="min-h-11 sm:min-h-9" disabled={isSubmitting}>
              {isSubmitting ? 'Registrando…' : copy.submit}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
