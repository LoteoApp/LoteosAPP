import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '../../../shared/ui/table'
import { formatCurrency } from '../../../shared/lib/formatCurrency'
import { formatDate } from '../../../shared/lib/formatDate'
import type { DownPayment, Installment } from '../types'
import InstallmentStateBadge from './InstallmentStateBadge'

type DebtInstallmentsTableProps = {
  entrega?: DownPayment
  cuotas: Installment[]
  currency: string
  // Absent when the statement is read-only (the venta isn't activa or the
  // table is being printed): no checkboxes are rendered then.
  selection?: {
    selected: readonly string[]
    includeDownPayment: boolean
    onToggleInstallment: (id: string) => void
    onToggleDownPayment: () => void
  }
}

export default function DebtInstallmentsTable({ entrega, cuotas, currency, selection }: DebtInstallmentsTableProps) {
  const selectable = selection !== undefined
  return (
    <div className="-mx-4 overflow-x-auto px-4 sm:mx-0 sm:px-0">
      <Table aria-label="Cuotas">
        <TableHeader>
          <TableRow>
            {selectable && <TableHead className="w-10">Cobrar</TableHead>}
            <TableHead className="w-24">Concepto</TableHead>
            <TableHead>Vencimiento</TableHead>
            <TableHead className="text-right">Monto</TableHead>
            <TableHead>Estado</TableHead>
            <TableHead>Pagada el</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {entrega !== undefined && (
            <TableRow>
              {selectable && (
                <TableCell>
                  {entrega.estado !== 'pagada' && (
                    <input
                      type="checkbox"
                      aria-label="Cobrar la entrega"
                      className="size-4 accent-primary"
                      checked={selection.includeDownPayment}
                      onChange={selection.onToggleDownPayment}
                    />
                  )}
                </TableCell>
              )}
              <TableCell>Entrega</TableCell>
              <TableCell className="text-muted-foreground">—</TableCell>
              <TableCell className="text-right tabular-nums">{formatCurrency(entrega.monto, currency)}</TableCell>
              <TableCell>
                <InstallmentStateBadge state={entrega.estado} />
              </TableCell>
              <TableCell>{entrega.fechaPago !== undefined ? formatDate(entrega.fechaPago) : '—'}</TableCell>
            </TableRow>
          )}
          {cuotas.map((cuota) => (
            <TableRow key={cuota.id}>
              {selectable && (
                <TableCell>
                  {cuota.estado !== 'pagada' && (
                    <input
                      type="checkbox"
                      aria-label={`Cobrar la cuota ${cuota.numero}`}
                      className="size-4 accent-primary"
                      checked={selection.selected.includes(cuota.id)}
                      onChange={() => selection.onToggleInstallment(cuota.id)}
                    />
                  )}
                </TableCell>
              )}
              <TableCell className="tabular-nums">Cuota {cuota.numero}</TableCell>
              <TableCell>{formatDate(cuota.fechaVencimiento)}</TableCell>
              <TableCell className="text-right tabular-nums">{formatCurrency(cuota.monto, currency)}</TableCell>
              <TableCell>
                <InstallmentStateBadge state={cuota.estado} />
              </TableCell>
              <TableCell>{cuota.fechaPago !== undefined ? formatDate(cuota.fechaPago) : '—'}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
