import { Input } from '../../../shared/ui/input'
import { Field, FieldLabel } from '../../../shared/ui/field'
import { Select, SelectContent, SelectItem, SelectList, SelectTrigger, SelectValue } from '../../../shared/ui/select'
import { SALE_STATES, SALE_STATE_LABELS, type SaleState } from '../types'

type SaleFiltersProps = {
  search: string
  state: SaleState | ''
  onSearchChange: (value: string) => void
  onStateChange: (value: SaleState | '') => void
}

export default function SaleFilters({ search, state, onSearchChange, onStateChange }: SaleFiltersProps) {
  return (
    <div className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_12rem]">
      <Field>
        <FieldLabel htmlFor="buscar-venta">Buscar</FieldLabel>
        <Input
          id="buscar-venta"
          type="search"
          placeholder="Cliente, loteo, lote o vendedor"
          value={search}
          onChange={(event) => onSearchChange(event.target.value)}
        />
      </Field>
      <Field>
        <FieldLabel htmlFor="estado-venta">Estado</FieldLabel>
        <Select
          value={state || 'todos'}
          onValueChange={(value) => onStateChange(value === 'todos' ? '' : (value as SaleState))}
        >
          <SelectTrigger id="estado-venta">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectList>
              <SelectItem value="todos">Todas</SelectItem>
              {SALE_STATES.map((candidate) => (
                <SelectItem key={candidate} value={candidate}>
                  {SALE_STATE_LABELS[candidate]}s
                </SelectItem>
              ))}
            </SelectList>
          </SelectContent>
        </Select>
      </Field>
    </div>
  )
}
