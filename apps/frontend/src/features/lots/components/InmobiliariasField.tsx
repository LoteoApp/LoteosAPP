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
import { listInmobiliarias, type Inmobiliaria } from '../api/list-inmobiliarias'

type InmobiliariasFieldProps = {
  selectedIds: readonly string[]
  onChange: (ids: string[]) => void
  inmobiliarias?: readonly Inmobiliaria[]
}

export default function InmobiliariasField({
  selectedIds,
  onChange,
  inmobiliarias = listInmobiliarias(),
}: InmobiliariasFieldProps) {
  const selected = inmobiliarias.filter((item) => selectedIds.includes(item.id))
  const allSelected = inmobiliarias.length > 0 && selected.length === inmobiliarias.length

  function handleValueChange(next: Inmobiliaria[]) {
    onChange(next.map((item) => item.id))
  }

  function handleToggleAll() {
    onChange(allSelected ? [] : inmobiliarias.map((item) => item.id))
  }

  return (
    <Field>
      <div className="flex items-center justify-between gap-2">
        <FieldLabel htmlFor="loteo-inmobiliarias">Inmobiliarias</FieldLabel>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="min-h-11 md:h-7 md:min-h-7"
          onClick={handleToggleAll}
          disabled={inmobiliarias.length === 0}
        >
          {allSelected ? 'Quitar todas' : 'Seleccionar todas'}
        </Button>
      </div>
      <Combobox
        items={[...inmobiliarias]}
        multiple
        value={[...selected]}
        onValueChange={handleValueChange}
        itemToStringLabel={(item) => item.razonSocial}
        itemToStringValue={(item) => item.id}
        isItemEqualToValue={(a, b) => a.id === b.id}
      >
        <ComboboxChips className="min-h-11 md:min-h-8">
          <ComboboxValue>
            {(value: Inmobiliaria[]) => (
              <>
                {value.map((item) => (
                  <ComboboxChip key={item.id} aria-label={item.razonSocial}>
                    {item.razonSocial}
                  </ComboboxChip>
                ))}
                <ComboboxChipsInput
                  id="loteo-inmobiliarias"
                  placeholder={value.length > 0 ? 'Buscar' : 'Buscar inmobiliaria'}
                  autoComplete="off"
                />
              </>
            )}
          </ComboboxValue>
        </ComboboxChips>
        <ComboboxContent>
          <ComboboxEmpty>No hay inmobiliarias con ese nombre.</ComboboxEmpty>
          <ComboboxList>
            {(item: Inmobiliaria) => (
              <ComboboxItem key={item.id} value={item} className="min-h-11 md:min-h-8">
                {item.razonSocial}
              </ComboboxItem>
            )}
          </ComboboxList>
        </ComboboxContent>
      </Combobox>
      <FieldDescription>Asigná una o varias inmobiliarias a este loteo.</FieldDescription>
    </Field>
  )
}
