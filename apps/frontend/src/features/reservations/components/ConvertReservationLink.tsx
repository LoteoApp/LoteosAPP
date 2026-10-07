import { BadgeDollarSign } from 'lucide-react'
import { Link } from 'react-router'
import { buttonVariants } from '../../../shared/ui/button'

type ConvertReservationLinkProps = {
  reservationId: string
  className?: string
  'aria-label'?: string
}

export default function ConvertReservationLink({
  reservationId,
  className = buttonVariants({ variant: 'outline' }),
  'aria-label': ariaLabel,
}: ConvertReservationLinkProps) {
  return (
    <Link to={`/reservas/${encodeURIComponent(reservationId)}/convertir`} className={className} aria-label={ariaLabel}>
      <BadgeDollarSign aria-hidden />
      Convertir en venta
    </Link>
  )
}
