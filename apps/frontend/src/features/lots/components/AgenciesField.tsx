import { Button } from '../../../shared/ui/button'
import {
  Combobox,
  ComboboxChip,
  ComboboxChips,
  ComboboxChipsInput,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxItem,
  ComboboxList,
  ComboboxValue,
} from '../../../shared/ui/combobox'
import { Field, FieldDescription, FieldLabel } from '../../../shared/ui/field'
import { listAgencies, type Agency } from '../api/list-agencies'

type AgenciesFieldProps = {
  selectedIds: readonly string[]
  onChange?: (ids: string[]) => void
  agencies?: readonly Agency[]
  disabled?: boolean
}

export default function AgenciesField({
  selectedIds,
  onChange,
  agencies = listAgencies(),
  disabled = false,
}: AgenciesFieldProps) {
  const selected = agencies.filter((item) => selectedIds.includes(item.id))
  const allSelected = agencies.length > 0 && selected.length === agencies.length

  function handleValueChange(next: Agency[]) {
    onChange?.(next.map((item) => item.id))
  }

  function handleToggleAll() {
    onChange?.(allSelected ? [] : agencies.map((item) => item.id))
  }

  return (
    <Field>
      <div className="flex items-center justify-between gap-2">
        <FieldLabel htmlFor="loteo-agencies">Inmobiliarias</FieldLabel>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="min-h-11 md:h-7 md:min-h-7"
          onClick={handleToggleAll}
          disabled={disabled || agencies.length === 0}
        >
          {allSelected ? 'Quitar todas' : 'Seleccionar todas'}
        </Button>
      </div>
      <Combobox
        items={[...agencies]}
        multiple
        value={[...selected]}
        onValueChange={handleValueChange}
        itemToStringLabel={(item) => item.businessName}
        itemToStringValue={(item) => item.id}
        isItemEqualToValue={(a, b) => a.id === b.id}
        disabled={disabled}
      >
        <ComboboxChips className="min-h-11 md:min-h-8">
          <ComboboxValue>
            {(value: Agency[]) => (
              <>
                {value.map((item) => (
                  <ComboboxChip key={item.id} aria-label={item.businessName}>
                    {item.businessName}
                  </ComboboxChip>
                ))}
                <ComboboxChipsInput
                  id="loteo-agencies"
                  placeholder={value.length > 0 ? 'Buscar' : 'Buscar inmobiliaria'}
                  autoComplete="off"
                  disabled={disabled}
                />
              </>
            )}
          </ComboboxValue>
        </ComboboxChips>
        <ComboboxContent>
          <ComboboxEmpty>No hay inmobiliarias con ese nombre.</ComboboxEmpty>
          <ComboboxList>
            {(item: Agency) => (
              <ComboboxItem key={item.id} value={item} className="min-h-11 md:min-h-8">
                {item.businessName}
              </ComboboxItem>
            )}
          </ComboboxList>
        </ComboboxContent>
      </Combobox>
      <FieldDescription>
        {disabled
          ? 'Disponible cuando el catálogo de inmobiliarias esté conectado.'
          : 'Asigná una o varias inmobiliarias a este loteo.'}
      </FieldDescription>
    </Field>
  )
}
