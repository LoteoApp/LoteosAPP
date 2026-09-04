import AgencyListItem from './AgencyListItem'
import type { Agency } from '../types'

type AgencyListProps = {
  agencies: Agency[]
  canManage: boolean
  isSubmitting: boolean
  onEdit: (id: string) => void
  onDeactivate: (id: string) => Promise<boolean>
}

export default function AgencyList({
  agencies,
  canManage,
  isSubmitting,
  onEdit,
  onDeactivate,
}: AgencyListProps) {
  return (
    <ul className="flex flex-col gap-3">
      {agencies.map((agency) => (
        <AgencyListItem
          key={agency.id}
          agency={agency}
          canManage={canManage}
          isSubmitting={isSubmitting}
          onEdit={() => onEdit(agency.id)}
          onDeactivate={() => onDeactivate(agency.id)}
        />
      ))}
    </ul>
  )
}
