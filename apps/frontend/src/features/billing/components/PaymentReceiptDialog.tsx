import { formatCurrency } from '../../../shared/lib/formatCurrency'
import { formatDate } from '../../../shared/lib/formatDate'
import {
  PAYMENT_MEDIUM_LABELS,
  PAYMENT_TYPE_LABELS,
  chargeLabel,
  clientLabel,
  lotLabel,
  paymentItemsLabel,
  type Payment,
  type StatementSale,
} from '../types'
import PrintDialog, { PrintField } from './PrintDialog'

type PaymentReceiptDialogProps = {
  open: boolean
  payment: Payment | null
  sale: StatementSale
  onClose: () => void
}

export default function PaymentReceiptDialog({ open, payment, sale, onClose }: PaymentReceiptDialogProps) {
  if (payment === null) {
    return null
  }
  return (
    <PrintDialog
      open={open}
      title="Recibo de cobro"
      description="Revisá los datos e imprimí el recibo o guardalo en PDF."
      documentTitle="Recibo de cobro"
      issuedAt={payment.fechaCreacion}
      printLabel="Imprimir recibo"
      onClose={onClose}
    >
      <div className="flex flex-col gap-3 border-b border-border px-4 py-5 sm:flex-row sm:items-end sm:justify-between sm:px-6 print:border-black">
        <div className="flex flex-col gap-1">
          <p className="text-[0.6875rem] font-medium tracking-[0.12em] text-muted-foreground uppercase print:text-black">
            Importe cobrado
          </p>
          <dl role="group" aria-label="Importe cobrado" className="flex flex-col gap-0.5">
            {payment.totales.map((total) => (
              <div key={total.moneda}>
                <dt className="sr-only">{total.moneda}</dt>
                <dd className="text-3xl font-semibold tracking-tight text-foreground tabular-nums print:text-black">
                  {formatCurrency(total.monto, total.moneda)}
                </dd>
              </div>
            ))}
          </dl>
        </div>
        <p className="text-sm font-medium text-foreground print:text-black">{PAYMENT_TYPE_LABELS[payment.tipo]}</p>
      </div>
      <dl className="grid grid-cols-1 gap-x-6 gap-y-5 px-4 py-5 sm:grid-cols-2 sm:px-6">
        <PrintField term="Concepto" className="sm:col-span-2">
          {paymentItemsLabel(payment)}
        </PrintField>
        <PrintField term="Lote">{lotLabel(sale)}</PrintField>
        <PrintField term="Fecha de pago">{formatDate(payment.fechaPago)}</PrintField>
        <PrintField term="Cliente">{clientLabel(sale.cliente)}</PrintField>
        <PrintField term="DNI">{sale.cliente.dni}</PrintField>
        <PrintField term="Medio de pago">{PAYMENT_MEDIUM_LABELS[payment.medioPago]}</PrintField>
        <PrintField term="Cobrado por">{clientLabel(payment.usuarioAlta)}</PrintField>
        {payment.observacion !== undefined && payment.observacion !== '' && (
          <PrintField term="Observación" className="sm:col-span-2">
            {payment.observacion}
          </PrintField>
        )}
      </dl>
      {(payment.cuotas.length > 0 || payment.incluyeEntrega) && (
        <table className="w-full border-t border-border text-sm print:border-black">
          <caption className="sr-only">Cuotas cobradas</caption>
          <thead>
            <tr className="text-left text-[0.6875rem] tracking-[0.12em] text-muted-foreground uppercase print:text-black">
              <th className="px-4 py-2 font-medium sm:px-6">Cuota</th>
              <th className="px-4 py-2 font-medium">Vencimiento</th>
              <th className="px-4 py-2 text-right font-medium sm:px-6">Monto</th>
            </tr>
          </thead>
          <tbody>
            {payment.incluyeEntrega && (
              <tr className="border-t border-border print:border-black">
                <td className="px-4 py-2 sm:px-6">Entrega</td>
                <td className="px-4 py-2">—</td>
                <td className="px-4 py-2 text-right tabular-nums sm:px-6">{formatCurrency(payment.montoEntrega, payment.moneda)}</td>
              </tr>
            )}
            {payment.cuotas.map((cuota) => (
              <tr key={cuota.id} className="border-t border-border print:border-black">
                <td className="px-4 py-2 sm:px-6">Cuota {cuota.numero}</td>
                <td className="px-4 py-2">{formatDate(cuota.fechaVencimiento)}</td>
                <td className="px-4 py-2 text-right tabular-nums sm:px-6">{formatCurrency(cuota.monto, payment.moneda)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {payment.cargos.length > 0 && (
        <table className="w-full border-t border-border text-sm print:border-black">
          <caption className="px-4 pt-3 text-left text-[0.6875rem] font-medium tracking-[0.12em] text-muted-foreground uppercase sm:px-6 print:text-black">
            Cargos adicionales
          </caption>
          <thead>
            <tr className="text-left text-[0.6875rem] tracking-[0.12em] text-muted-foreground uppercase print:text-black">
              <th className="px-4 py-2 font-medium sm:px-6">Concepto</th>
              <th className="px-4 py-2 text-right font-medium sm:px-6">Monto</th>
            </tr>
          </thead>
          <tbody>
            {payment.cargos.map((cargo) => (
              <tr key={cargo.id} className="border-t border-border print:border-black">
                <td className="px-4 py-2 sm:px-6">{chargeLabel(cargo)}</td>
                <td className="px-4 py-2 text-right tabular-nums sm:px-6">{formatCurrency(cargo.monto, cargo.moneda)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      <div className="grid grid-cols-1 gap-6 border-t border-border px-4 py-5 sm:grid-cols-2 sm:gap-10 sm:px-6 print:border-black">
        <SignatureLine label="Firma del cliente" />
        <SignatureLine label="Firma de quien cobra" />
      </div>
    </PrintDialog>
  )
}

function SignatureLine({ label }: { label: string }) {
  return (
    <div className="flex flex-col">
      <div className="h-12 border-b border-border print:border-black" />
      <p className="pt-2 text-xs text-muted-foreground print:text-black">{label}</p>
    </div>
  )
}
