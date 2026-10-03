import { Input } from '../../../shared/ui/input'
import { Field, FieldLabel } from '../../../shared/ui/field'
import { Select, SelectContent, SelectItem, SelectList, SelectTrigger, SelectValue } from '../../../shared/ui/select'
import { RESERVATION_STATES, type ReservationState } from '../types'

type StateFilter = ReservationState | 'todos'

const STATE_FILTER_LABELS: Record<StateFilter, string> = {
  todos: 'Todos',
  activa: 'Activas',
  vencida: 'Vencidas',
  cancelada: 'Canceladas',
  convertida: 'Convertidas',
}

type Props = {
  search: string
  state: ReservationState | ''
  onSearchChange: (value: string) => void
  onStateChange: (value: ReservationState | '') => void
}

export default function ReservationFilters({ search, state, onSearchChange, onStateChange }: Props) {
  return (
    <div className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_12rem]">
      <Field>
        <FieldLabel htmlFor="buscar-reserva">Buscar</FieldLabel>
        <Input
          id="buscar-reserva"
          type="search"
          placeholder="Cliente, loteo, lote o vendedor"
          value={search}
          onChange={(event) => onSearchChange(event.target.value)}
        />
      </Field>
      <Field>
        <FieldLabel htmlFor="estado-reserva">Estado</FieldLabel>
        <Select value={state || 'todos'} onValueChange={(value) => onStateChange(value === 'todos' ? '' : value as ReservationState)}>
          <SelectTrigger id="estado-reserva">
            <SelectValue>{(current: StateFilter) => STATE_FILTER_LABELS[current]}</SelectValue>
          </SelectTrigger>
          <SelectContent>
            <SelectList>
              {(['todos', ...RESERVATION_STATES] as const).map((candidate) => (
                <SelectItem key={candidate} value={candidate}>
                  {STATE_FILTER_LABELS[candidate]}
                </SelectItem>
              ))}
            </SelectList>
          </SelectContent>
        </Select>
      </Field>
    </div>
  )
}
