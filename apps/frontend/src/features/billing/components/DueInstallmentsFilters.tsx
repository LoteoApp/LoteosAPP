import { Input } from '../../../shared/ui/input'
import { Field, FieldLabel } from '../../../shared/ui/field'
import { Select, SelectContent, SelectItem, SelectList, SelectTrigger, SelectValue } from '../../../shared/ui/select'
import { DUE_FILTERS, DUE_FILTER_LABELS, type DueFilter } from '../types'

export type DueFilterValues = {
  search: string
  state: DueFilter
  from: string
  to: string
}

type DueInstallmentsFiltersProps = {
  values: DueFilterValues
  onChange: (values: DueFilterValues) => void
}

export default function DueInstallmentsFilters({ values, onChange }: DueInstallmentsFiltersProps) {
  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-[minmax(0,1fr)_10rem_10rem_10rem]">
      <Field className="sm:col-span-2 lg:col-span-1">
        <FieldLabel htmlFor="buscar-cuota">Buscar</FieldLabel>
        <Input
          id="buscar-cuota"
          type="search"
          placeholder="Cliente, DNI, loteo, lote o vendedor"
          value={values.search}
          onChange={(event) => onChange({ ...values, search: event.target.value })}
        />
      </Field>
      <Field>
        <FieldLabel htmlFor="estado-cuota">Estado</FieldLabel>
        <Select
          value={values.state}
          onValueChange={(value) => onChange({ ...values, state: value as DueFilter })}
        >
          <SelectTrigger id="estado-cuota">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectList>
              {DUE_FILTERS.map((candidate) => (
                <SelectItem key={candidate} value={candidate}>
                  {DUE_FILTER_LABELS[candidate]}
                </SelectItem>
              ))}
            </SelectList>
          </SelectContent>
        </Select>
      </Field>
      <Field>
        <FieldLabel htmlFor="vence-desde">Vence desde</FieldLabel>
        <Input
          id="vence-desde"
          type="date"
          value={values.from}
          onChange={(event) => onChange({ ...values, from: event.target.value })}
        />
      </Field>
      <Field>
        <FieldLabel htmlFor="vence-hasta">Vence hasta</FieldLabel>
        <Input
          id="vence-hasta"
          type="date"
          value={values.to}
          onChange={(event) => onChange({ ...values, to: event.target.value })}
        />
      </Field>
    </div>
  )
}
