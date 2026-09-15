import { Badge } from '../../../shared/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '../../../shared/ui/table'
import { formatArea } from '../../../shared/lib/formatArea'
import { formatCurrency } from '../../../shared/lib/formatCurrency'
import { formatDate } from '../../../shared/lib/formatDate'
import { formatDateTime } from '../../../shared/lib/formatDateTime'
import { formatPercent } from '../../../shared/lib/formatPercent'
import {
  INSTALLMENT_STATE_LABELS,
  PAYMENT_METHOD_LABELS,
  PAYMENT_PERIOD_LABELS,
  type Installment,
  type PaymentPlan,
  type Sale,
} from '../types'
import SaleStatusBadge from './SaleStatusBadge'

export default function SaleDetails({ sale }: { sale: Sale }) {
  const loteLabel = sale.loteNumero ? `Lote ${sale.loteNumero}` : 'Lote sin número'
  return (
    <div className="flex flex-col gap-4">
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
      {sale.planPago !== undefined && <PaymentPlanCard plan={sale.planPago} />}
    </div>
  )
}

function PaymentPlanCard({ plan }: { plan: PaymentPlan }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Plan de pago</CardTitle>
        <CardDescription>
          {plan.cantidadCuotas} {plan.cantidadCuotas === 1 ? 'cuota' : 'cuotas'} ·{' '}
          {PAYMENT_PERIOD_LABELS[plan.periodicidad].toLowerCase()}
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <dl className="grid gap-3 text-sm sm:grid-cols-2">
          {plan.montoEntrega > 0 && (
            <Info label="Entrega" value={formatCurrency(plan.montoEntrega, plan.moneda)} />
          )}
          <Info label="Monto financiado" value={formatCurrency(plan.montoFinanciado, plan.moneda)} />
          <Info label="Tasa de interés" value={formatPercent(plan.tasaInteres)} />
          <Info label="Monto por cuota" value={formatCurrency(plan.montoCuota, plan.moneda)} />
          <Info label="Total financiado" value={formatCurrency(plan.montoTotal, plan.moneda)} />
        </dl>
        {plan.cuotas !== undefined && plan.cuotas.length > 0 && (
          <InstallmentsTable cuotas={plan.cuotas} moneda={plan.moneda} />
        )}
      </CardContent>
    </Card>
  )
}

function InstallmentsTable({ cuotas, moneda }: { cuotas: Installment[]; moneda: string }) {
  return (
    <div className="-mx-4 overflow-x-auto px-4 sm:mx-0 sm:px-0">
      <Table aria-label="Cuotas">
        <TableHeader>
          <TableRow>
            <TableHead className="w-16">N.º</TableHead>
            <TableHead>Vencimiento</TableHead>
            <TableHead className="text-right">Monto</TableHead>
            <TableHead>Estado</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {cuotas.map((cuota) => (
            <TableRow key={cuota.id}>
              <TableCell className="tabular-nums">{cuota.numero}</TableCell>
              <TableCell>{formatDate(cuota.fechaVencimiento)}</TableCell>
              <TableCell className="text-right tabular-nums">{formatCurrency(cuota.monto, moneda)}</TableCell>
              <TableCell>
                <Badge variant={cuota.estado === 'pagada' ? 'default' : 'outline'}>
                  {INSTALLMENT_STATE_LABELS[cuota.estado]}
                </Badge>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
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
