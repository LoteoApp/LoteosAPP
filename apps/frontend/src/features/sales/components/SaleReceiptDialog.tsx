import { formatArea } from '../../../shared/lib/formatArea'
import { formatCurrency } from '../../../shared/lib/formatCurrency'
import { formatPercent } from '../../../shared/lib/formatPercent'
import {
  ReceiptDialog,
  ReceiptField,
  ReceiptFields,
  ReceiptSection,
  ReceiptSignatures,
  ReceiptSummary,
} from '../../../shared/ui/receipt'
import { installmentsLabel } from './installmentsLabel'
import {
  PAYMENT_METHOD_LABELS,
  PAYMENT_PERIOD_LABELS,
  lotOptionLabel,
  sellerAgencyLabel,
  sellerOptionLabel,
  type SaleReceipt,
  type SaleReceiptPlan,
} from '../types'

type SaleReceiptDialogProps = {
  open: boolean
  receipt: SaleReceipt | null
  onClose: () => void
}

function PlanSummary({ plan, currency }: { plan: SaleReceiptPlan; currency: string }) {
  return (
    <ReceiptSection title="Plan de pago">
      <ReceiptFields label="Plan de pago">
        {plan.downPayment > 0 && <ReceiptField term="Entrega">{formatCurrency(plan.downPayment, currency)}</ReceiptField>}
        <ReceiptField term="Monto financiado">{formatCurrency(plan.financedAmount, currency)}</ReceiptField>
        <ReceiptField term="Cuotas">
          {installmentsLabel(plan.installments, plan.installmentAmount, currency)} ·{' '}
          {PAYMENT_PERIOD_LABELS[plan.period].toLowerCase()}
        </ReceiptField>
        <ReceiptField term="Tasa de interés">{formatPercent(plan.interestRate)}</ReceiptField>
        <ReceiptField term="Total financiado">{formatCurrency(plan.totalAmount, currency)}</ReceiptField>
      </ReceiptFields>
    </ReceiptSection>
  )
}

export default function SaleReceiptDialog({ open, receipt, onClose }: SaleReceiptDialogProps) {
  if (receipt === null) {
    return null
  }

  const { lot, client, seller, method, amount, currency, issuedAt, plan } = receipt

  return (
    <ReceiptDialog
      open={open}
      title="Venta confirmada"
      description="Revisá los datos y generá el recibo para imprimirlo o guardarlo en PDF."
      documentTitle="Recibo de venta"
      issuedAt={issuedAt}
      printLabel="Imprimir recibo"
      onClose={onClose}
    >
      <ReceiptSummary label="Total de la operación" asideLabel="Modalidad" aside={PAYMENT_METHOD_LABELS[method]}>
        {formatCurrency(amount, currency)}
      </ReceiptSummary>
      <ReceiptSection title="Datos de la venta">
        <ReceiptFields>
          <ReceiptField term="Lote" wide={lot.area === null}>
            {lotOptionLabel(lot)}
          </ReceiptField>
          {lot.area !== null && <ReceiptField term="Superficie">{formatArea(lot.area)}</ReceiptField>}
          <ReceiptField term="Comprador">
            {client.apellido}, {client.nombre}
          </ReceiptField>
          <ReceiptField term="DNI">{client.dni}</ReceiptField>
          <ReceiptField term="Vendedor">{sellerOptionLabel(seller)}</ReceiptField>
          <ReceiptField term="Inmobiliaria">{sellerAgencyLabel(seller)}</ReceiptField>
        </ReceiptFields>
      </ReceiptSection>
      {plan !== undefined && <PlanSummary plan={plan} currency={currency} />}
      <ReceiptSignatures labels={['Firma del comprador', 'Firma del vendedor']} />
    </ReceiptDialog>
  )
}
