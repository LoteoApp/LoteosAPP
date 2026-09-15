import { Link } from 'react-router'
import { Button } from '../../../shared/ui/button'
import { Card, CardContent } from '../../../shared/ui/card'
import { formatCurrency } from '../../../shared/lib/formatCurrency'
import { formatDateTime } from '../../../shared/lib/formatDateTime'
import { sellerAgencyLabel, sellerOptionLabel, type Sale } from '../types'
import SaleStatusBadge from './SaleStatusBadge'

function sellerOf(sale: Sale) {
  return {
    ...sale.vendedor,
    inmobiliariaId: sale.inmobiliaria?.id,
    inmobiliariaRazonSocial: sale.inmobiliaria?.razonSocial,
  }
}

export default function SalesList({ sales }: { sales: Sale[] }) {
  if (sales.length === 0) {
    return <p className="text-sm text-muted-foreground">No hay ventas que coincidan con los filtros.</p>
  }
  return (
    <ul className="grid gap-3">
      {sales.map((sale) => {
        const seller = sellerOf(sale)
        const loteLabel = sale.loteNumero ? `Lote ${sale.loteNumero}` : 'Lote sin número'
        return (
          <li key={sale.id}>
            <Card>
              <CardContent className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center">
                <div className="grid gap-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <Link to={`/ventas/${sale.id}`} className="font-medium hover:underline">
                      {sale.loteoNombre}
                    </Link>
                    <Link
                      to={`/lotes/${sale.loteoId}`}
                      className="text-sm text-muted-foreground hover:underline"
                      aria-label={`Ver loteo ${sale.loteoNombre}`}
                    >
                      Ver loteo
                    </Link>
                    <SaleStatusBadge state={sale.estado} />
                  </div>
                  <p className="text-sm">
                    {sale.manzanaNumero ? `Mz ${sale.manzanaNumero} · ` : ''}
                    {loteLabel} · {sale.cliente.apellido}, {sale.cliente.nombre}
                  </p>
                  <p className="text-sm text-muted-foreground">
                    {formatCurrency(sale.monto, sale.moneda)} · {sellerOptionLabel(seller)} ({sellerAgencyLabel(seller)})
                  </p>
                  <p className="text-sm text-muted-foreground">Registrada: {formatDateTime(sale.fechaCreacion)}</p>
                </div>
                <div className="flex flex-wrap gap-2 sm:justify-end">
                  <Button
                    render={(
                      <Link
                        to={`/ventas/${sale.id}`}
                        aria-label={`Ver detalle de la venta del ${sale.loteNumero ? `lote ${sale.loteNumero}` : 'lote sin número'} en ${sale.loteoNombre}`}
                      />
                    )}
                  >
                    Ver detalle
                  </Button>
                </div>
              </CardContent>
            </Card>
          </li>
        )
      })}
    </ul>
  )
}
