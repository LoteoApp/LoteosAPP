import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from '../../../shared/ui/combobox'
import { Field, FieldDescription, FieldLabel } from '../../../shared/ui/field'
import { formatArea } from '../../../shared/lib/formatArea'
import { formatCurrency } from '../../../shared/lib/formatCurrency'
import { loteOptionLabel, type LoteOption } from '../types'

type LoteComboboxProps = {
  lotes: readonly LoteOption[]
  value: LoteOption | null
  onChange: (lote: LoteOption | null) => void
  isLoading?: boolean
  disabled?: boolean
}

function describe(lote: LoteOption): string {
  const parts: string[] = []
  if (lote.superficie !== null) {
    parts.push(formatArea(lote.superficie))
  }
  if (lote.precio !== null) {
    parts.push(formatCurrency(lote.precio, lote.moneda))
  }

  return parts.join(' · ')
}

export default function LoteCombobox({
  lotes,
  value,
  onChange,
  isLoading = false,
  disabled = false,
}: LoteComboboxProps) {
  return (
    <Field>
      <FieldLabel htmlFor="venta-lote">Lote</FieldLabel>
      <Combobox
        items={[...lotes]}
        value={value}
        onValueChange={onChange}
        itemToStringLabel={loteOptionLabel}
        itemToStringValue={(item: LoteOption) => item.id}
        isItemEqualToValue={(a: LoteOption, b: LoteOption) => a.id === b.id}
        disabled={disabled || isLoading}
      >
        <ComboboxInput
          id="venta-lote"
          placeholder={isLoading ? 'Cargando lotes…' : 'Buscar por loteo, manzana o lote'}
          autoComplete="off"
          showClear
          className="min-h-11 md:min-h-9"
          disabled={disabled || isLoading}
        />
        <ComboboxContent>
          <ComboboxEmpty>No hay lotes disponibles con ese criterio.</ComboboxEmpty>
          <ComboboxList>
            {(item: LoteOption) => (
              <ComboboxItem key={item.id} value={item} className="min-h-11 md:min-h-9">
                <span className="flex flex-col">
                  <span>{loteOptionLabel(item)}</span>
                  {describe(item) !== '' && (
                    <span className="text-xs text-muted-foreground">{describe(item)}</span>
                  )}
                </span>
              </ComboboxItem>
            )}
          </ComboboxList>
        </ComboboxContent>
      </Combobox>
      <FieldDescription>Solo se ofrecen lotes disponibles para la venta.</FieldDescription>
    </Field>
  )
}
