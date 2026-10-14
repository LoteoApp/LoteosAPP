import { formatCurrency } from '../../../shared/lib/formatCurrency'
import { formatDate } from '../../../shared/lib/formatDate'
import { formatPercent } from '../../../shared/lib/formatPercent'
import {
  ReceiptDialog,
  ReceiptField,
  ReceiptFields,
  ReceiptSection,
  ReceiptSummary,
  ReceiptTable,
  ReceiptTableCell,
  ReceiptTableRow,
} from '../../../shared/ui/receipt'
import {
  INSTALLMENT_STATE_LABELS,
  PAYMENT_METHOD_LABELS,
  PAYMENT_PERIOD_LABELS,
  clientLabel,
  lotLabel,
  type DebtStatement,
} from '../types'

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
    <ReceiptDialog
      open={open}
      title="Estado de deuda"
      description="Imprimí el estado de deuda o guardalo en PDF para entregárselo al cliente."
      documentTitle="Estado de deuda"
      issuedAt={statement.emitidoEl}
      printLabel="Imprimir estado de deuda"
      onClose={onClose}
    >
      <ReceiptSummary
        label="Saldo pendiente"
        asideLabel="Vencido"
        aside={formatCurrency(resumen.montoVencido, currency)}
      >
        {formatCurrency(resumen.montoPendiente, currency)}
      </ReceiptSummary>
      <ReceiptSection title="Datos de la venta">
        <ReceiptFields>
          <ReceiptField term="Cliente">{clientLabel(venta.cliente)}</ReceiptField>
          <ReceiptField term="DNI">{venta.cliente.dni}</ReceiptField>
          <ReceiptField term="Lote">{lotLabel(venta)}</ReceiptField>
          <ReceiptField term="Modalidad">{PAYMENT_METHOD_LABELS[venta.modalidadPago] ?? venta.modalidadPago}</ReceiptField>
          <ReceiptField term="Precio de venta">{formatCurrency(venta.monto, currency)}</ReceiptField>
          <ReceiptField term="Fecha de venta">{formatDate(venta.fechaCreacion)}</ReceiptField>
          {plan !== undefined && (
            <>
              <ReceiptField term="Plan">
                {plan.cantidadCuotas} {plan.cantidadCuotas === 1 ? 'cuota' : 'cuotas'}{' '}
                {PAYMENT_PERIOD_LABELS[plan.periodicidad] ?? plan.periodicidad} de{' '}
                {formatCurrency(plan.montoCuota, currency)}
              </ReceiptField>
              <ReceiptField term="Tasa de interés">{formatPercent(plan.tasaInteres)}</ReceiptField>
            </>
          )}
          <ReceiptField term="Total del plan">{formatCurrency(resumen.montoTotal, currency)}</ReceiptField>
          <ReceiptField term="Pagado">{formatCurrency(resumen.montoPagado, currency)}</ReceiptField>
          {statement.cargosCobrados.length > 0 && (
            <ReceiptField term="Cargos adicionales cobrados" wide>
              {statement.cargosCobrados.map((total) => formatCurrency(total.monto, total.moneda)).join(' · ')}
            </ReceiptField>
          )}
        </ReceiptFields>
      </ReceiptSection>
      <ReceiptTable
        caption="Detalle de cuotas"
        columns={[
          { label: 'Concepto' },
          { label: 'Vencimiento' },
          { label: 'Monto', numeric: true },
          { label: 'Estado' },
          { label: 'Pagada el' },
        ]}
      >
        {entrega !== undefined && (
          <ReceiptTableRow>
            <ReceiptTableCell>Entrega</ReceiptTableCell>
            <ReceiptTableCell>—</ReceiptTableCell>
            <ReceiptTableCell numeric>{formatCurrency(entrega.monto, currency)}</ReceiptTableCell>
            <ReceiptTableCell>{INSTALLMENT_STATE_LABELS[entrega.estado]}</ReceiptTableCell>
            <ReceiptTableCell>{entrega.fechaPago !== undefined ? formatDate(entrega.fechaPago) : '—'}</ReceiptTableCell>
          </ReceiptTableRow>
        )}
        {cuotas.map((cuota) => (
          <ReceiptTableRow key={cuota.id}>
            <ReceiptTableCell>Cuota {cuota.numero}</ReceiptTableCell>
            <ReceiptTableCell>{formatDate(cuota.fechaVencimiento)}</ReceiptTableCell>
            <ReceiptTableCell numeric>{formatCurrency(cuota.monto, currency)}</ReceiptTableCell>
            <ReceiptTableCell>{INSTALLMENT_STATE_LABELS[cuota.estado]}</ReceiptTableCell>
            <ReceiptTableCell>{cuota.fechaPago !== undefined ? formatDate(cuota.fechaPago) : '—'}</ReceiptTableCell>
          </ReceiptTableRow>
        ))}
      </ReceiptTable>
    </ReceiptDialog>
  )
}
