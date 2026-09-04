import { Label } from '../../../shared/ui/label'
import type { LoteoManzana } from '../types'

export const ALL_MANZANAS = ''

type ManzanaFilterProps = {
  manzanas: LoteoManzana[]
  value: string
  onChange: (manzanaId: string) => void
}

export default function ManzanaFilter({ manzanas, value, onChange }: ManzanaFilterProps) {
  if (manzanas.length <= 1) {
    return null
  }

  return (
    <div className="flex flex-col gap-1.5">
      <Label htmlFor="filtro-manzana">Manzana</Label>
      <select
        id="filtro-manzana"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="h-9 w-full min-w-0 rounded-lg border border-input bg-transparent px-3 py-1 text-base outline-none transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 md:text-sm sm:max-w-xs dark:bg-input/30"
      >
        <option value={ALL_MANZANAS}>Todas las manzanas</option>
        {manzanas.map((manzana) => (
          <option key={manzana.id} value={manzana.id}>
            {manzana.numero || 'Sin número'}
          </option>
        ))}
      </select>
    </div>
  )
}
