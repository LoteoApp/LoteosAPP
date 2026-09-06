import { Button } from '../../../shared/ui/button'
import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from '../../../shared/ui/combobox'
import { Field, FieldLabel } from '../../../shared/ui/field'
import { clienteOptionLabel, type ClienteOption } from '../types'

type ClientComboboxProps = {
  clientes: readonly ClienteOption[]
  value: ClienteOption | null
  onChange: (cliente: ClienteOption | null) => void
  onRegisterClient: () => void
  isLoading?: boolean
  disabled?: boolean
}

export default function ClientCombobox({
  clientes,
  value,
  onChange,
  onRegisterClient,
  isLoading = false,
  disabled = false,
}: ClientComboboxProps) {
  return (
    <Field>
      <div className="flex items-center justify-between gap-2">
        <FieldLabel htmlFor="venta-cliente">Cliente</FieldLabel>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="min-h-11 md:h-7 md:min-h-7"
          onClick={onRegisterClient}
          disabled={disabled}
        >
          Registrar cliente
        </Button>
      </div>
      <Combobox
        items={[...clientes]}
        value={value}
        onValueChange={onChange}
        itemToStringLabel={clienteOptionLabel}
        itemToStringValue={(item: ClienteOption) => item.id}
        isItemEqualToValue={(a: ClienteOption, b: ClienteOption) => a.id === b.id}
        disabled={disabled || isLoading}
      >
        <ComboboxInput
          id="venta-cliente"
          placeholder={isLoading ? 'Cargando clientes…' : 'Buscar por nombre, apellido o DNI'}
          autoComplete="off"
          showClear
          className="min-h-11 md:min-h-9"
          disabled={disabled || isLoading}
        />
        <ComboboxContent>
          <ComboboxEmpty>
            No hay clientes con ese criterio. Podés registrarlo con «Registrar cliente».
          </ComboboxEmpty>
          <ComboboxList>
            {(item: ClienteOption) => (
              <ComboboxItem key={item.id} value={item} className="min-h-11 md:min-h-9">
                {clienteOptionLabel(item)}
              </ComboboxItem>
            )}
          </ComboboxList>
        </ComboboxContent>
      </Combobox>
    </Field>
  )
}
