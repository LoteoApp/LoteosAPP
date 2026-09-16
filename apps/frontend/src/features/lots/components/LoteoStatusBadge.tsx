import { CircleCheck, Clock } from 'lucide-react'
import { Badge } from '../../../shared/ui/badge'

type LoteoStatusBadgeProps = {
  hasPlan: boolean
}

export default function LoteoStatusBadge({ hasPlan }: LoteoStatusBadgeProps) {
  if (hasPlan) {
    return (
      <Badge variant="secondary">
        <CircleCheck aria-hidden />
        Plano cargado
      </Badge>
    )
  }

  return (
    <Badge variant="outline" className="text-muted-foreground">
      <Clock aria-hidden />
      Plano pendiente
    </Badge>
  )
}
