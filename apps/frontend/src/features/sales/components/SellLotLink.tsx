import { BadgeDollarSign } from 'lucide-react'
import { Link } from 'react-router'
import { Button, buttonVariants } from '../../../shared/ui/button'
import { saleDisabledReason, type SaleableLot } from '../types'

type SellLotLinkProps = {
  developmentId: string
  lot: SaleableLot & { id: string }
}

// The viewer's entry point to a sale: a link to the sale page for this lote,
// or a disabled button explaining what the lote is missing.
export default function SellLotLink({ developmentId, lot }: SellLotLinkProps) {
  const disabledReason = saleDisabledReason(lot)
  if (disabledReason) {
    return (
      <span title={disabledReason} className="inline-flex">
        <Button type="button" variant="outline" disabled>
          <BadgeDollarSign aria-hidden />
          Pasar a venta
        </Button>
      </span>
    )
  }
  return (
    <Link
      className={buttonVariants({ variant: 'outline' })}
      to={`/ventas/nueva/${encodeURIComponent(developmentId)}/${encodeURIComponent(lot.id)}`}
    >
      <BadgeDollarSign aria-hidden />
      Pasar a venta
    </Link>
  )
}
