import { Label } from '../../../shared/ui/label'
import { Select, SelectContent, SelectItem, SelectList, SelectTrigger, SelectValue } from '../../../shared/ui/select'
import { SearchField } from '../../../shared/ui/search-field'
import { GESTIONABLE_ROLES, ROLE_LABELS, type GestionableRol } from '../types'

export type RolFilter = 'todos' | GestionableRol
export type EstadoFilter = 'todos' | 'activos' | 'pendientes' | 'inactivos'

const ESTADO_FILTER_LABELS: Record<EstadoFilter, string> = {
  todos: 'Todos',
  activos: 'Activos',
  pendientes: 'Invitación pendiente',
  inactivos: 'Dados de baja',
}

const ESTADO_FILTERS = Object.keys(ESTADO_FILTER_LABELS) as EstadoFilter[]

function rolFilterLabel(filter: RolFilter) {
  return filter === 'todos' ? 'Todos' : ROLE_LABELS[filter]
}

type UsersFiltersProps = {
  search: string
  rolFilter: RolFilter
  estadoFilter: EstadoFilter
  onSearchChange: (search: string) => void
  onRolFilterChange: (filter: RolFilter) => void
  onEstadoFilterChange: (filter: EstadoFilter) => void
}

export default function UsersFilters({
  search,
  rolFilter,
  estadoFilter,
  onSearchChange,
  onRolFilterChange,
  onEstadoFilterChange,
}: UsersFiltersProps) {
  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:flex-wrap">
      <SearchField
        id="user-search"
        placeholder="Nombre, apellido o correo"
        value={search}
        onChange={onSearchChange}
        inputClassName="sm:w-64"
      />
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="filtro-rol">Rol</Label>
        <Select
          value={rolFilter}
          onValueChange={(value) => onRolFilterChange(value as RolFilter)}
        >
          <SelectTrigger id="filtro-rol" className="sm:w-48">
            <SelectValue>{(current: RolFilter) => rolFilterLabel(current)}</SelectValue>
          </SelectTrigger>
          <SelectContent>
            <SelectList>
              {(['todos', ...GESTIONABLE_ROLES] as const).map((candidate) => (
                <SelectItem key={candidate} value={candidate}>
                  {rolFilterLabel(candidate)}
                </SelectItem>
              ))}
            </SelectList>
          </SelectContent>
        </Select>
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="filtro-estado">Estado</Label>
        <Select
          value={estadoFilter}
          onValueChange={(value) => onEstadoFilterChange(value as EstadoFilter)}
        >
          <SelectTrigger id="filtro-estado" className="sm:w-48">
            <SelectValue>{(current: EstadoFilter) => ESTADO_FILTER_LABELS[current]}</SelectValue>
          </SelectTrigger>
          <SelectContent>
            <SelectList>
              {ESTADO_FILTERS.map((candidate) => (
                <SelectItem key={candidate} value={candidate}>
                  {ESTADO_FILTER_LABELS[candidate]}
                </SelectItem>
              ))}
            </SelectList>
          </SelectContent>
        </Select>
      </div>
    </div>
  )
}
