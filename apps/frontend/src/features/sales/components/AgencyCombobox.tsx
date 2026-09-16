import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from '../../../shared/ui/combobox'
import { Field, FieldDescription, FieldLabel } from '../../../shared/ui/field'
import type { AgencyOption } from '../types'

type AgencyComboboxProps = {
  agencies: readonly AgencyOption[]
  value: AgencyOption | null
  onChange: (agency: AgencyOption | null) => void
  isLoading?: boolean
  disabled?: boolean
}

export default function AgencyCombobox({
  agencies,
  value,
  onChange,
  isLoading = false,
  disabled = false,
}: AgencyComboboxProps) {
  return (
    <Field>
      <FieldLabel htmlFor="venta-inmobiliaria">Inmobiliaria</FieldLabel>
      <Combobox
        items={[...agencies]}
        value={value}
        onValueChange={onChange}
        itemToStringLabel={(item: AgencyOption) => item.razonSocial}
        itemToStringValue={(item: AgencyOption) => item.id}
        isItemEqualToValue={(a: AgencyOption, b: AgencyOption) => a.id === b.id}
        disabled={disabled || isLoading}
      >
        <ComboboxInput
          id="venta-inmobiliaria"
          placeholder={isLoading ? 'Cargando inmobiliarias…' : 'Buscar inmobiliaria'}
          autoComplete="off"
          showClear
          className="min-h-11 md:min-h-9"
          disabled={disabled || isLoading}
        />
        <ComboboxContent>
          <ComboboxEmpty>No hay inmobiliarias con ese nombre.</ComboboxEmpty>
          <ComboboxList>
            {(item: AgencyOption) => (
              <ComboboxItem key={item.id} value={item} className="min-h-11 md:min-h-9">
                {item.razonSocial}
              </ComboboxItem>
            )}
          </ComboboxList>
        </ComboboxContent>
      </Combobox>
      <FieldDescription>Inmobiliaria del vendedor, o venta directa.</FieldDescription>
    </Field>
  )
}
