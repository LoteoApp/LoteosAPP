import { Plus, Trash2 } from 'lucide-react'
import { Button } from '../../../shared/ui/button'
import { Field, FieldDescription, FieldLabel } from '../../../shared/ui/field'
import { Input } from '../../../shared/ui/input'
import { Select, SelectContent, SelectItem, SelectList, SelectTrigger, SelectValue } from '../../../shared/ui/select'
import {
  CHARGE_TYPES,
  CHARGE_TYPE_LABELS,
  MAX_CHARGE_DETAIL_LENGTH,
  MAX_PAYMENT_CHARGES,
  currencyOptions,
  newChargeRow,
  type ChargeRow,
  type ChargeType,
} from '../types'

type ChargesEditorProps = {
  rows: ChargeRow[]
  // The sale's currency: what a new charge starts in, though each row may be
  // changed to another one.
  currency: string
  onChange: (rows: ChargeRow[]) => void
}

export default function ChargesEditor({ rows, currency, onChange }: ChargesEditorProps) {
  const currencies = currencyOptions(currency)

  function update(key: string, changes: Partial<ChargeRow>) {
    onChange(rows.map((row) => (row.key === key ? { ...row, ...changes } : row)))
  }

  return (
    <fieldset className="flex flex-col gap-3 rounded-lg border border-border px-4 py-3">
      <legend className="px-1 text-sm font-medium">Cargos adicionales</legend>
      <FieldDescription>
        Impuestos, gastos administrativos, honorarios o servicios que se cobran junto con la cuota. Cada cargo puede
        estar en otra moneda: se cobra aparte, no se convierte.
      </FieldDescription>
      {rows.map((row, index) => (
        <div key={row.key} className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_7rem_7rem_auto] sm:items-end">
          <Field>
            <FieldLabel htmlFor={`cargo-tipo-${row.key}`}>Tipo</FieldLabel>
            <Select
              value={row.tipo}
              onValueChange={(value) => update(row.key, { tipo: value as ChargeType })}
            >
              <SelectTrigger id={`cargo-tipo-${row.key}`}>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectList>
                  {CHARGE_TYPES.map((candidate) => (
                    <SelectItem key={candidate} value={candidate}>
                      {CHARGE_TYPE_LABELS[candidate]}
                    </SelectItem>
                  ))}
                </SelectList>
              </SelectContent>
            </Select>
          </Field>
          <Field>
            <FieldLabel htmlFor={`cargo-monto-${row.key}`}>Monto</FieldLabel>
            <Input
              id={`cargo-monto-${row.key}`}
              inputMode="decimal"
              placeholder="0,00"
              autoComplete="off"
              value={row.monto}
              onChange={(event) => update(row.key, { monto: event.target.value })}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor={`cargo-moneda-${row.key}`}>Moneda</FieldLabel>
            <Select value={row.moneda} onValueChange={(value) => update(row.key, { moneda: value as string })}>
              <SelectTrigger id={`cargo-moneda-${row.key}`}>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectList>
                  {currencies.map((candidate) => (
                    <SelectItem key={candidate} value={candidate}>
                      {candidate}
                    </SelectItem>
                  ))}
                </SelectList>
              </SelectContent>
            </Select>
          </Field>
          <Button
            type="button"
            variant="ghost"
            className="min-h-11 w-fit sm:min-h-9"
            onClick={() => onChange(rows.filter((candidate) => candidate.key !== row.key))}
            aria-label={`Quitar el cargo ${index + 1}`}
          >
            <Trash2 aria-hidden />
            Quitar
          </Button>
          <Field className="sm:col-span-4">
            <FieldLabel htmlFor={`cargo-detalle-${row.key}`}>Detalle</FieldLabel>
            <Input
              id={`cargo-detalle-${row.key}`}
              maxLength={MAX_CHARGE_DETAIL_LENGTH}
              placeholder="Período, comprobante, etc."
              autoComplete="off"
              value={row.detalle}
              onChange={(event) => update(row.key, { detalle: event.target.value })}
            />
          </Field>
        </div>
      ))}
      <Button
        type="button"
        variant="outline"
        className="min-h-11 w-fit sm:min-h-9"
        disabled={rows.length >= MAX_PAYMENT_CHARGES}
        onClick={() => onChange([...rows, newChargeRow(currency)])}
      >
        <Plus aria-hidden />
        Agregar cargo
      </Button>
    </fieldset>
  )
}
