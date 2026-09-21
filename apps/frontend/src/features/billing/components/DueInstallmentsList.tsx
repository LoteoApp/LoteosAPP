import { Link } from 'react-router'
import { Button } from '../../../shared/ui/button'
import { Card, CardContent } from '../../../shared/ui/card'
import { formatCurrency } from '../../../shared/lib/formatCurrency'
import { formatDate } from '../../../shared/lib/formatDate'
import { clientLabel, lotLabel, type DueInstallment, type DueInstallmentPage } from '../types'
import InstallmentStateBadge from './InstallmentStateBadge'

export function DueInstallmentsList({ cuotas }: { cuotas: DueInstallment[] }) {
  if (cuotas.length === 0) {
    return <p className="text-sm text-muted-foreground">No hay cuotas que coincidan con los filtros.</p>
  }
  return (
    <ul className="grid gap-3">
      {cuotas.map((cuota) => (
        <li key={cuota.id}>
          <Card>
            <CardContent className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center">
              <div className="grid gap-1">
                <div className="flex flex-wrap items-center gap-2">
                  <Link to={`/cobranzas/${cuota.ventaId}`} className="font-medium hover:underline">
                    {clientLabel(cuota.cliente)}
                  </Link>
                  <InstallmentStateBadge state={cuota.estado} />
                </div>
                <p className="text-sm">{lotLabel(cuota)}</p>
                <p className="text-sm text-muted-foreground">
                  Cuota {cuota.numero} de {cuota.cantidadCuotas} · {formatCurrency(cuota.monto, cuota.moneda)} · Vence{' '}
                  {formatDate(cuota.fechaVencimiento)}
                  {cuota.fechaPago !== undefined ? ` · Pagada el ${formatDate(cuota.fechaPago)}` : ''}
                </p>
                <p className="text-sm text-muted-foreground">
                  Vendedor: {clientLabel(cuota.vendedor)}
                  {cuota.inmobiliaria ? ` (${cuota.inmobiliaria.razonSocial})` : ' (Venta directa)'}
                </p>
              </div>
              <div className="flex flex-wrap gap-2 sm:justify-end">
                <Button
                  render={
                    <Link
                      to={`/cobranzas/${cuota.ventaId}`}
                      aria-label={`Ver estado de deuda de ${clientLabel(cuota.cliente)} por ${lotLabel(cuota)}, cuota ${cuota.numero}`}
                    />
                  }
                >
                  Estado de deuda
                </Button>
              </div>
            </CardContent>
          </Card>
        </li>
      ))}
    </ul>
  )
}

type DueInstallmentsPaginationProps = {
  page: Pick<DueInstallmentPage, 'pagina' | 'paginas' | 'total'>
  isLoading: boolean
  onPageChange: (page: number) => void
}

export function DueInstallmentsPagination({ page, isLoading, onPageChange }: DueInstallmentsPaginationProps) {
  if (page.paginas <= 1) {
    return null
  }
  return (
    <nav className="flex flex-wrap items-center justify-between gap-3" aria-label="Paginación de cuotas">
      <p className="text-sm text-muted-foreground">
        Página {page.pagina} de {page.paginas} · {page.total} cuotas
      </p>
      <div className="flex gap-2">
        <Button
          type="button"
          variant="outline"
          disabled={page.pagina <= 1 || isLoading}
          onClick={() => onPageChange(Math.max(1, page.pagina - 1))}
        >
          Anterior
        </Button>
        <Button
          type="button"
          variant="outline"
          disabled={page.pagina >= page.paginas || isLoading}
          onClick={() => onPageChange(Math.min(page.paginas, page.pagina + 1))}
        >
          Siguiente
        </Button>
      </div>
    </nav>
  )
}
