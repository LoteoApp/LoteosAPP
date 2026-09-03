import { Label } from '../../../shared/ui/label'
import { Select, SelectContent, SelectItem, SelectList, SelectTrigger, SelectValue } from '../../../shared/ui/select'
import { GESTIONABLE_ROLES, ROLE_LABELS, type GestionableRol } from '../types'

export type RolFilter = 'todos' | GestionableRol
export type EstadoFilter = 'todos' | 'activos' | 'inactivos'

type UsersFiltersProps = {
  rolFilter: RolFilter
  estadoFilter: EstadoFilter
  onRolFilterChange: (filter: RolFilter) => void
  onEstadoFilterChange: (filter: EstadoFilter) => void
}

export default function UsersFilters({
  rolFilter,
  estadoFilter,
  onRolFilterChange,
  onEstadoFilterChange,
}: UsersFiltersProps) {
  return (
    <div className="flex flex-col gap-3 sm:flex-row">
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="filtro-rol">Rol</Label>
        <Select
          value={rolFilter}
          onValueChange={(value) => onRolFilterChange(value as RolFilter)}
        >
          <SelectTrigger id="filtro-rol" className="sm:w-48">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectList>
              <SelectItem value="todos">Todos</SelectItem>
              {GESTIONABLE_ROLES.map((candidate) => (
                <SelectItem key={candidate} value={candidate}>
                  {ROLE_LABELS[candidate]}
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
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectList>
              <SelectItem value="todos">Todos</SelectItem>
              <SelectItem value="activos">Activos</SelectItem>
              <SelectItem value="inactivos">Dados de baja</SelectItem>
            </SelectList>
          </SelectContent>
        </Select>
      </div>
    </div>
  )
}
