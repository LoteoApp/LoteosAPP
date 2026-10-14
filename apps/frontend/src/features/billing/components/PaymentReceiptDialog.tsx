import { formatCurrency } from '../../../shared/lib/formatCurrency'
import { formatDate } from '../../../shared/lib/formatDate'
import {
  ReceiptDialog,
  ReceiptField,
  ReceiptFields,
  ReceiptSection,
  ReceiptSignatures,
  ReceiptSummary,
  ReceiptTable,
  ReceiptTableCell,
  ReceiptTableRow,
} from '../../../shared/ui/receipt'
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
    <ReceiptDialog
      open={open}
      title="Recibo de cobro"
      description="Revisá los datos e imprimí el recibo o guardalo en PDF."
      documentTitle="Recibo de cobro"
      issuedAt={payment.fechaCreacion}
      printLabel="Imprimir recibo"
      onClose={onClose}
    >
      <ReceiptSummary label="Importe cobrado" asideLabel="Tipo de cobro" aside={PAYMENT_TYPE_LABELS[payment.tipo]}>
        <dl role="group" aria-label="Importe cobrado" className="flex flex-col gap-0.5">
          {payment.totales.map((total) => (
            <div key={total.moneda}>
              <dt className="sr-only">{total.moneda}</dt>
              <dd>{formatCurrency(total.monto, total.moneda)}</dd>
            </div>
          ))}
        </dl>
      </ReceiptSummary>
      <ReceiptSection title="Datos del cobro">
        <ReceiptFields>
          <ReceiptField term="Concepto" wide>
            {paymentItemsLabel(payment)}
          </ReceiptField>
          <ReceiptField term="Lote">{lotLabel(sale)}</ReceiptField>
          <ReceiptField term="Fecha de pago">{formatDate(payment.fechaPago)}</ReceiptField>
          <ReceiptField term="Cliente">{clientLabel(sale.cliente)}</ReceiptField>
          <ReceiptField term="DNI">{sale.cliente.dni}</ReceiptField>
          <ReceiptField term="Medio de pago">{PAYMENT_MEDIUM_LABELS[payment.medioPago]}</ReceiptField>
          <ReceiptField term="Cobrado por">{clientLabel(payment.usuarioAlta)}</ReceiptField>
          {payment.observacion !== undefined && payment.observacion !== '' && (
            <ReceiptField term="Observación" wide>
              {payment.observacion}
            </ReceiptField>
          )}
        </ReceiptFields>
      </ReceiptSection>
      {(payment.cuotas.length > 0 || payment.incluyeEntrega) && (
        <ReceiptTable
          caption="Cuotas cobradas"
          columns={[{ label: 'Cuota' }, { label: 'Vencimiento' }, { label: 'Monto', numeric: true }]}
        >
          {payment.incluyeEntrega && (
            <ReceiptTableRow>
              <ReceiptTableCell>Entrega</ReceiptTableCell>
              <ReceiptTableCell>—</ReceiptTableCell>
              <ReceiptTableCell numeric>{formatCurrency(payment.montoEntrega, payment.moneda)}</ReceiptTableCell>
            </ReceiptTableRow>
          )}
          {payment.cuotas.map((cuota) => (
            <ReceiptTableRow key={cuota.id}>
              <ReceiptTableCell>Cuota {cuota.numero}</ReceiptTableCell>
              <ReceiptTableCell>{formatDate(cuota.fechaVencimiento)}</ReceiptTableCell>
              <ReceiptTableCell numeric>{formatCurrency(cuota.monto, payment.moneda)}</ReceiptTableCell>
            </ReceiptTableRow>
          ))}
        </ReceiptTable>
      )}
      {payment.cargos.length > 0 && (
        <ReceiptTable caption="Cargos adicionales" columns={[{ label: 'Concepto' }, { label: 'Monto', numeric: true }]}>
          {payment.cargos.map((cargo) => (
            <ReceiptTableRow key={cargo.id}>
              <ReceiptTableCell>{chargeLabel(cargo)}</ReceiptTableCell>
              <ReceiptTableCell numeric>{formatCurrency(cargo.monto, cargo.moneda)}</ReceiptTableCell>
            </ReceiptTableRow>
          ))}
        </ReceiptTable>
      )}
      <ReceiptSignatures labels={['Firma del cliente', 'Firma de quien cobra']} />
    </ReceiptDialog>
  )
}
