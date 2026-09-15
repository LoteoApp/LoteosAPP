import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from '../../../shared/ui/combobox'
import { Field, FieldDescription, FieldLabel } from '../../../shared/ui/field'
import { sellerOptionLabel, type SellerOption } from '../types'

type SellerComboboxProps = {
  sellers: readonly SellerOption[]
  value: SellerOption | null
  onChange: (seller: SellerOption | null) => void
  isLoading?: boolean
  disabled?: boolean
  description?: string
}

export default function SellerCombobox({
  sellers,
  value,
  onChange,
  isLoading = false,
  disabled = false,
  description = 'Usuario responsable de la venta.',
}: SellerComboboxProps) {
  const isDisabled = disabled || isLoading

  return (
    <Field>
      <FieldLabel htmlFor="venta-vendedor">Vendedor</FieldLabel>
      <Combobox
        items={[...sellers]}
        value={value}
        onValueChange={onChange}
        itemToStringLabel={(item: SellerOption) => sellerOptionLabel(item)}
        itemToStringValue={(item: SellerOption) => item.id}
        isItemEqualToValue={(a: SellerOption, b: SellerOption) => a.id === b.id}
        disabled={isDisabled}
      >
        <ComboboxInput
          id="venta-vendedor"
          placeholder={isLoading ? 'Cargando vendedores…' : 'Buscar vendedor'}
          autoComplete="off"
          showClear
          className="min-h-11 md:min-h-9"
          disabled={isDisabled}
        />
        <ComboboxContent>
          <ComboboxEmpty>No hay vendedores con ese nombre.</ComboboxEmpty>
          <ComboboxList>
            {(item: SellerOption) => (
              <ComboboxItem key={item.id} value={item} className="min-h-11 md:min-h-9">
                {sellerOptionLabel(item)}
              </ComboboxItem>
            )}
          </ComboboxList>
        </ComboboxContent>
      </Combobox>
      <FieldDescription>{description}</FieldDescription>
    </Field>
  )
}
