import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card'
import { formatArea } from '../../../shared/lib/formatArea'
import { formatCurrency } from '../../../shared/lib/formatCurrency'
import { formatDateTime } from '../../../shared/lib/formatDateTime'
import { PAYMENT_METHOD_LABELS, type Sale } from '../types'
import SaleStatusBadge from './SaleStatusBadge'

export default function SaleDetails({ sale }: { sale: Sale }) {
  const loteLabel = sale.loteNumero ? `Lote ${sale.loteNumero}` : 'Lote sin número'
  return (
    <Card>
      <CardHeader>
        <div className="flex flex-wrap items-center gap-2">
          <CardTitle>{sale.loteoNombre}</CardTitle>
          <SaleStatusBadge state={sale.estado} />
        </div>
        <CardDescription>
          {sale.manzanaNumero ? `Manzana ${sale.manzanaNumero} · ` : ''}
          {loteLabel}
          {sale.loteSuperficie !== null ? ` · ${formatArea(sale.loteSuperficie)}` : ''}
        </CardDescription>
      </CardHeader>
      <CardContent className="grid gap-3 text-sm sm:grid-cols-2">
        <Info label="Monto" value={formatCurrency(sale.monto, sale.moneda)} />
        <Info label="Modalidad de pago" value={PAYMENT_METHOD_LABELS[sale.modalidadPago]} />
        <Info label="Comprador" value={`${sale.cliente.apellido}, ${sale.cliente.nombre} · DNI ${sale.cliente.dni}`} />
        <Info label="Vendedor" value={`${sale.vendedor.apellido}, ${sale.vendedor.nombre}`} />
        <Info label="Inmobiliaria" value={sale.inmobiliaria ? sale.inmobiliaria.razonSocial : 'Venta directa'} />
        <Info label="Cargada por" value={`${sale.usuarioAlta.apellido}, ${sale.usuarioAlta.nombre}`} />
        <Info label="Registrada" value={formatDateTime(sale.fechaCreacion)} />
      </CardContent>
    </Card>
  )
}

function Info({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs font-medium text-muted-foreground">{label}</dt>
      <dd>{value}</dd>
    </div>
  )
}
