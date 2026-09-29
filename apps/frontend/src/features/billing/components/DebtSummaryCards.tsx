import { Card, CardContent } from '../../../shared/ui/card'
import { formatCurrency } from '../../../shared/lib/formatCurrency'
import { formatDate } from '../../../shared/lib/formatDate'
import type { DebtSummary } from '../types'

type DebtSummaryCardsProps = {
  summary: DebtSummary
  currency: string
}

export default function DebtSummaryCards({ summary, currency }: DebtSummaryCardsProps) {
  const pendingDetail =
    summary.cuotasPendientes === 0
      ? 'Sin cuotas pendientes'
      : `${summary.cuotasPendientes} ${summary.cuotasPendientes === 1 ? 'cuota pendiente' : 'cuotas pendientes'}`
  const nextDue =
    summary.proximoVencimiento !== undefined ? `Próximo vencimiento: ${formatDate(summary.proximoVencimiento)}` : ''
  return (
    <dl className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <SummaryCard label="Total del plan" value={formatCurrency(summary.montoTotal, currency)} />
      <SummaryCard
        label="Pagado"
        value={formatCurrency(summary.montoPagado, currency)}
        detail={`${summary.cuotasPagadas} ${summary.cuotasPagadas === 1 ? 'cuota pagada' : 'cuotas pagadas'}`}
      />
      <SummaryCard label="Saldo pendiente" value={formatCurrency(summary.montoPendiente, currency)} detail={pendingDetail} />
      <SummaryCard
        label="Vencido"
        value={formatCurrency(summary.montoVencido, currency)}
        detail={summary.cuotasVencidas > 0 ? `${summary.cuotasVencidas} ${summary.cuotasVencidas === 1 ? 'cuota vencida' : 'cuotas vencidas'}` : nextDue}
        highlight={summary.cuotasVencidas > 0}
      />
    </dl>
  )
}

function SummaryCard({
  label,
  value,
  detail,
  highlight = false,
}: {
  label: string
  value: string
  detail?: string
  highlight?: boolean
}) {
  return (
    <Card>
      <CardContent className="flex flex-col gap-1">
        <dt className="text-xs font-medium text-muted-foreground">{label}</dt>
        <dd className={`text-xl font-semibold tabular-nums ${highlight ? 'text-destructive' : ''}`}>{value}</dd>
        {detail !== undefined && detail !== '' && <dd className="text-xs text-muted-foreground">{detail}</dd>}
      </CardContent>
    </Card>
  )
}
