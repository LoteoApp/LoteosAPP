import { BadgeDollarSign } from 'lucide-react'
import { Link } from 'react-router'
import { Button, buttonVariants } from '../../../shared/ui/button'
import { saleDisabledReason } from '../types'

type SellLotLinkProps = {
  loteoId: string
  lote: { id: string; numero: string; precio: number | null }
}

// The viewer's entry point to a sale: a link to the sale page for this lote,
// or a disabled button explaining what the lote is missing.
export default function SellLotLink({ loteoId, lote }: SellLotLinkProps) {
  const disabledReason = saleDisabledReason(lote)
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
      to={`/ventas/nueva/${encodeURIComponent(loteoId)}/${encodeURIComponent(lote.id)}`}
    >
      <BadgeDollarSign aria-hidden />
      Pasar a venta
    </Link>
  )
}
