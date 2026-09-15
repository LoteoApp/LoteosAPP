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
import { clientOptionLabel, type ClientOption } from '../types'

type ClientComboboxProps = {
  clients: readonly ClientOption[]
  value: ClientOption | null
  onChange: (client: ClientOption | null) => void
  onRegisterClient: () => void
  isLoading?: boolean
  disabled?: boolean
}

export default function ClientCombobox({
  clients,
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
        items={[...clients]}
        value={value}
        onValueChange={onChange}
        itemToStringLabel={clientOptionLabel}
        itemToStringValue={(item: ClientOption) => item.id}
        isItemEqualToValue={(a: ClientOption, b: ClientOption) => a.id === b.id}
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
            {(item: ClientOption) => (
              <ComboboxItem key={item.id} value={item} className="min-h-11 md:min-h-9">
                {clientOptionLabel(item)}
              </ComboboxItem>
            )}
          </ComboboxList>
        </ComboboxContent>
      </Combobox>
    </Field>
  )
}
