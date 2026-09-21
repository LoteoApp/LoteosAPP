import { useState } from 'react'
import { Alert, AlertDescription } from '../../../shared/ui/alert'
import { Card, CardContent } from '../../../shared/ui/card'
import DueInstallmentsFilters, { type DueFilterValues } from '../components/DueInstallmentsFilters'
import { DueInstallmentsList, DueInstallmentsPagination } from '../components/DueInstallmentsList'
import { useDueInstallments } from '../hooks/use-due-installments'
import type { DueSummary } from '../types'

type BillingPageProps = { accessToken?: string }

const INITIAL_FILTERS: DueFilterValues = { search: '', state: 'pendientes', from: '', to: '' }

export default function BillingPage({ accessToken = '' }: BillingPageProps) {
  const [filters, setFilters] = useState<DueFilterValues>(INITIAL_FILTERS)
  const [pageNumber, setPageNumber] = useState(1)
  const due = useDueInstallments(accessToken, {
    q: filters.search || undefined,
    estado: filters.state,
    desde: filters.from || undefined,
    hasta: filters.to || undefined,
    pagina: pageNumber,
  })

  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-foreground">Cobranzas</h1>
          <p className="text-sm text-muted-foreground">
            Consultá los vencimientos de las ventas financiadas y registrá los cobros desde el estado de deuda de cada venta.
          </p>
        </div>
      </div>
      <DueSummaryCards summary={due.page.resumen} isLoading={due.isLoading} />
      <section className="grid gap-3">
        <DueInstallmentsFilters
          values={filters}
          onChange={(next) => {
            setFilters(next)
            setPageNumber(1)
          }}
        />
        {due.error && (
          <Alert variant="destructive">
            <AlertDescription>{due.error}</AlertDescription>
          </Alert>
        )}
        {due.isLoading ? (
          <p role="status" className="text-sm text-muted-foreground">
            Cargando cuotas…
          </p>
        ) : (
          <DueInstallmentsList cuotas={due.page.cuotas} />
        )}
        <DueInstallmentsPagination page={due.page} isLoading={due.isLoading} onPageChange={setPageNumber} />
      </section>
    </section>
  )
}

function DueSummaryCards({ summary, isLoading }: { summary: DueSummary; isLoading: boolean }) {
  const overdue = isLoading ? '…' : String(summary.cuotasVencidas)
  const soon = isLoading ? '…' : String(summary.cuotasProximas)
  return (
    <dl className="grid gap-3 sm:grid-cols-2">
      <Card>
        <CardContent className="flex flex-col gap-1">
          <dt className="text-xs font-medium text-muted-foreground">Cuotas vencidas</dt>
          <dd className="text-2xl font-semibold tabular-nums text-destructive">{overdue}</dd>
        </CardContent>
      </Card>
      <Card>
        <CardContent className="flex flex-col gap-1">
          <dt className="text-xs font-medium text-muted-foreground">Vencen en los próximos 30 días</dt>
          <dd className="text-2xl font-semibold tabular-nums">{soon}</dd>
        </CardContent>
      </Card>
    </dl>
  )
}
