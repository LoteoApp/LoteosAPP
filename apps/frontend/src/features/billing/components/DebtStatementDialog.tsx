import { formatCurrency } from '../../../shared/lib/formatCurrency'
import { formatDate } from '../../../shared/lib/formatDate'
import { formatPercent } from '../../../shared/lib/formatPercent'
import {
  INSTALLMENT_STATE_LABELS,
  PAYMENT_METHOD_LABELS,
  PAYMENT_PERIOD_LABELS,
  clientLabel,
  lotLabel,
  type DebtStatement,
} from '../types'
import PrintDialog, { PrintField } from './PrintDialog'

type DebtStatementDialogProps = {
  open: boolean
  statement: DebtStatement
  onClose: () => void
}

export default function DebtStatementDialog({ open, statement, onClose }: DebtStatementDialogProps) {
  const { venta, entrega, cuotas, resumen } = statement
  const currency = venta.moneda
  const plan = venta.planPago
  return (
    <PrintDialog
      open={open}
      title="Estado de deuda"
      description="Imprimí el estado de deuda o guardalo en PDF para entregárselo al cliente."
      documentTitle="Estado de deuda"
      issuedAt={statement.emitidoEl}
      printLabel="Imprimir estado de deuda"
      onClose={onClose}
    >
      <div className="flex flex-col gap-3 border-b border-border px-4 py-5 sm:flex-row sm:items-end sm:justify-between sm:px-6 print:border-black">
        <div className="flex flex-col gap-1">
          <p className="text-[0.6875rem] font-medium tracking-[0.12em] text-muted-foreground uppercase print:text-black">
            Saldo pendiente
          </p>
          <p className="text-3xl font-semibold tracking-tight text-foreground tabular-nums print:text-black">
            {formatCurrency(resumen.montoPendiente, currency)}
          </p>
        </div>
        <div className="flex flex-col gap-1 sm:items-end">
          <p className="text-[0.6875rem] font-medium tracking-[0.12em] text-muted-foreground uppercase print:text-black">
            Vencido
          </p>
          <p className="text-lg font-semibold tabular-nums text-foreground print:text-black">
            {formatCurrency(resumen.montoVencido, currency)}
          </p>
        </div>
      </div>
      <dl className="grid grid-cols-1 gap-x-6 gap-y-5 px-4 py-5 sm:grid-cols-2 sm:px-6">
        <PrintField term="Cliente">{clientLabel(venta.cliente)}</PrintField>
        <PrintField term="DNI">{venta.cliente.dni}</PrintField>
        <PrintField term="Lote">{lotLabel(venta)}</PrintField>
        <PrintField term="Modalidad">{PAYMENT_METHOD_LABELS[venta.modalidadPago] ?? venta.modalidadPago}</PrintField>
        <PrintField term="Precio de venta">{formatCurrency(venta.monto, currency)}</PrintField>
        <PrintField term="Fecha de venta">{formatDate(venta.fechaCreacion)}</PrintField>
        {plan !== undefined && (
          <>
            <PrintField term="Plan">
              {plan.cantidadCuotas} {plan.cantidadCuotas === 1 ? 'cuota' : 'cuotas'}{' '}
              {PAYMENT_PERIOD_LABELS[plan.periodicidad] ?? plan.periodicidad} de{' '}
              {formatCurrency(plan.montoCuota, currency)}
            </PrintField>
            <PrintField term="Tasa de interés">{formatPercent(plan.tasaInteres)}</PrintField>
          </>
        )}
        <PrintField term="Total del plan">{formatCurrency(resumen.montoTotal, currency)}</PrintField>
        <PrintField term="Pagado">{formatCurrency(resumen.montoPagado, currency)}</PrintField>
        {statement.cargosCobrados.length > 0 && (
          <PrintField term="Cargos adicionales cobrados" className="sm:col-span-2">
            {statement.cargosCobrados.map((total) => formatCurrency(total.monto, total.moneda)).join(' · ')}
          </PrintField>
        )}
      </dl>
      <table className="w-full border-t border-border text-sm print:border-black">
        <caption className="sr-only">Detalle de cuotas</caption>
        <thead>
          <tr className="text-left text-[0.6875rem] tracking-[0.12em] text-muted-foreground uppercase print:text-black">
            <th className="px-4 py-2 font-medium sm:px-6">Concepto</th>
            <th className="px-4 py-2 font-medium">Vencimiento</th>
            <th className="px-4 py-2 text-right font-medium">Monto</th>
            <th className="px-4 py-2 font-medium">Estado</th>
            <th className="px-4 py-2 font-medium sm:px-6">Pagada el</th>
          </tr>
        </thead>
        <tbody>
          {entrega !== undefined && (
            <tr className="border-t border-border print:border-black">
              <td className="px-4 py-2 sm:px-6">Entrega</td>
              <td className="px-4 py-2">—</td>
              <td className="px-4 py-2 text-right tabular-nums">{formatCurrency(entrega.monto, currency)}</td>
              <td className="px-4 py-2">{INSTALLMENT_STATE_LABELS[entrega.estado]}</td>
              <td className="px-4 py-2 sm:px-6">{entrega.fechaPago !== undefined ? formatDate(entrega.fechaPago) : '—'}</td>
            </tr>
          )}
          {cuotas.map((cuota) => (
            <tr key={cuota.id} className="border-t border-border print:border-black">
              <td className="px-4 py-2 sm:px-6">Cuota {cuota.numero}</td>
              <td className="px-4 py-2">{formatDate(cuota.fechaVencimiento)}</td>
              <td className="px-4 py-2 text-right tabular-nums">{formatCurrency(cuota.monto, currency)}</td>
              <td className="px-4 py-2">{INSTALLMENT_STATE_LABELS[cuota.estado]}</td>
              <td className="px-4 py-2 sm:px-6">{cuota.fechaPago !== undefined ? formatDate(cuota.fechaPago) : '—'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </PrintDialog>
  )
}
