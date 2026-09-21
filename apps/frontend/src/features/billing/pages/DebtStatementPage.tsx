import { useState } from 'react'
import { BadgeDollarSign, CheckCheck, Printer } from 'lucide-react'
import { Link, useParams } from 'react-router'
import { messageFromError } from '../../../shared/api/client'
import { Alert, AlertDescription, AlertTitle } from '../../../shared/ui/alert'
import { Badge } from '../../../shared/ui/badge'
import { Button, buttonVariants } from '../../../shared/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card'
import { formatCurrency } from '../../../shared/lib/formatCurrency'
import { formatDate } from '../../../shared/lib/formatDate'
import { registerPayment, settleSale } from '../api/billing'
import DebtInstallmentsTable from '../components/DebtInstallmentsTable'
import DebtStatementDialog from '../components/DebtStatementDialog'
import DebtSummaryCards from '../components/DebtSummaryCards'
import PaymentDialog from '../components/PaymentDialog'
import PaymentReceiptDialog from '../components/PaymentReceiptDialog'
import PaymentsHistory from '../components/PaymentsHistory'
import { useDebtStatement } from '../hooks/use-debt-statement'
import {
  PAYMENT_METHOD_LABELS,
  SALE_STATE_LABELS,
  clientLabel,
  lotLabel,
  paymentItemsLabel,
  selectionAmount,
  toggleInstallment,
  type DebtStatement,
  type Payment,
  type PaymentTerms,
  type StatementSale,
} from '../types'

type DebtStatementPageProps = { accessToken?: string }

type OpenDialog = { kind: 'none' } | { kind: 'pago' } | { kind: 'cancelacion_total' } | { kind: 'estado' } | { kind: 'recibo'; payment: Payment }

export default function DebtStatementPage({ accessToken = '' }: DebtStatementPageProps) {
  const { ventaId = '' } = useParams()
  const debt = useDebtStatement(accessToken, ventaId)
  const [selected, setSelected] = useState<string[]>([])
  const [includeDownPayment, setIncludeDownPayment] = useState(false)
  const [dialog, setDialog] = useState<OpenDialog>({ kind: 'none' })
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState<string | null>(null)

  const statement = debt.statement
  const collectable = statement !== null && statement.venta.estado === 'activa'
  const amount = statement === null ? 0 : selectionAmount(statement, selected, includeDownPayment)
  const hasSelection = includeDownPayment || selected.length > 0

  async function submit(run: () => Promise<Payment>) {
    setIsSubmitting(true)
    setSubmitError(null)
    try {
      const payment = await run()
      setSelected([])
      setIncludeDownPayment(false)
      setDialog({ kind: 'recibo', payment })
      debt.refresh()
    } catch (error: unknown) {
      setSubmitError(messageFromError(error))
    } finally {
      setIsSubmitting(false)
    }
  }

  function handlePayment(terms: PaymentTerms) {
    void submit(() =>
      registerPayment(accessToken, ventaId, { ...terms, cuotaIds: selected, incluirEntrega: includeDownPayment }),
    )
  }

  function handleSettlement(terms: PaymentTerms) {
    if (statement === null) return
    void submit(() => settleSale(accessToken, ventaId, { ...terms, montoEsperado: statement.resumen.montoPendiente }))
  }

  function closeDialog() {
    setDialog({ kind: 'none' })
    setSubmitError(null)
  }

  return (
    <section className="flex flex-col gap-4">
      <Link to="/cobranzas" className="w-fit text-sm text-muted-foreground hover:text-foreground">
        Volver a cobranzas
      </Link>
      {debt.isLoading && (
        <p role="status" className="text-sm text-muted-foreground">
          Cargando estado de deuda…
        </p>
      )}
      {debt.error && (
        <Alert variant="destructive">
          <AlertTitle>No se pudo cargar el estado de deuda</AlertTitle>
          <AlertDescription>{debt.error}</AlertDescription>
        </Alert>
      )}
      {!debt.isLoading && statement !== null && (
        <>
          <SaleCard sale={statement.venta} />
          <DebtSummaryCards summary={statement.resumen} currency={statement.venta.moneda} />
          <Card>
            <CardHeader>
              <CardTitle>Cuotas</CardTitle>
              <CardDescription>
                {collectable
                  ? 'Marcá lo que el cliente paga: las cuotas se cobran en orden, de la más antigua a la más nueva.'
                  : `La venta está ${SALE_STATE_LABELS[statement.venta.estado]?.toLowerCase() ?? statement.venta.estado}; no admite nuevos cobros.`}
              </CardDescription>
            </CardHeader>
            <CardContent className="flex flex-col gap-4">
              <DebtInstallmentsTable
                entrega={statement.entrega}
                cuotas={statement.cuotas}
                currency={statement.venta.moneda}
                selection={
                  collectable
                    ? {
                        selected,
                        includeDownPayment,
                        onToggleInstallment: (id) => setSelected(toggleInstallment(statement.cuotas, selected, id)),
                        onToggleDownPayment: () => setIncludeDownPayment((value) => !value),
                      }
                    : undefined
                }
              />
              <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center">
                {collectable && (
                  <>
                    <Button
                      type="button"
                      className="min-h-11 sm:min-h-9"
                      disabled={!hasSelection}
                      onClick={() => setDialog({ kind: 'pago' })}
                    >
                      <BadgeDollarSign aria-hidden />
                      Registrar cobro
                      {hasSelection ? ` · ${formatCurrency(amount, statement.venta.moneda)}` : ''}
                    </Button>
                    <Button
                      type="button"
                      variant="outline"
                      className="min-h-11 sm:min-h-9"
                      disabled={statement.resumen.montoPendiente <= 0}
                      onClick={() => setDialog({ kind: 'cancelacion_total' })}
                    >
                      <CheckCheck aria-hidden />
                      Cancelar saldo total
                    </Button>
                  </>
                )}
                <Button
                  type="button"
                  variant="ghost"
                  className="min-h-11 sm:min-h-9"
                  onClick={() => setDialog({ kind: 'estado' })}
                >
                  <Printer aria-hidden />
                  Imprimir estado de deuda
                </Button>
                <Link className={buttonVariants({ variant: 'ghost' })} to={`/ventas/${statement.venta.id}`}>
                  Ver venta
                </Link>
              </div>
            </CardContent>
          </Card>
          <PaymentsHistory cobros={statement.cobros} onPrint={(payment) => setDialog({ kind: 'recibo', payment })} />
          {(dialog.kind === 'pago' || dialog.kind === 'cancelacion_total') && (
            <PaymentDialog
              open
              mode={dialog.kind}
              amount={dialog.kind === 'pago' ? amount : statement.resumen.montoPendiente}
              currency={statement.venta.moneda}
              itemsLabel={dialog.kind === 'pago' ? selectionLabel(statement, selected, includeDownPayment) : 'Todo el saldo pendiente'}
              isSubmitting={isSubmitting}
              error={submitError}
              onSubmit={dialog.kind === 'pago' ? handlePayment : handleSettlement}
              onClose={closeDialog}
            />
          )}
          <DebtStatementDialog open={dialog.kind === 'estado'} statement={statement} onClose={closeDialog} />
          <PaymentReceiptDialog
            open={dialog.kind === 'recibo'}
            payment={dialog.kind === 'recibo' ? dialog.payment : null}
            sale={statement.venta}
            onClose={closeDialog}
          />
        </>
      )}
    </section>
  )
}

function selectionLabel(statement: DebtStatement, selected: readonly string[], includeDownPayment: boolean): string {
  return paymentItemsLabel({
    incluyeEntrega: includeDownPayment,
    cuotas: statement.cuotas.filter((cuota) => selected.includes(cuota.id)),
  })
}

function SaleCard({ sale }: { sale: StatementSale }) {
  return (
    <Card>
      <CardHeader>
        <div className="flex flex-wrap items-center gap-2">
          <CardTitle>{clientLabel(sale.cliente)}</CardTitle>
          <Badge variant={sale.estado === 'activa' ? 'default' : 'outline'}>
            {SALE_STATE_LABELS[sale.estado] ?? sale.estado}
          </Badge>
        </div>
        <CardDescription>
          {lotLabel(sale)} · DNI {sale.cliente.dni}
        </CardDescription>
      </CardHeader>
      <CardContent className="grid gap-3 text-sm sm:grid-cols-2 lg:grid-cols-4">
        <Info label="Precio de venta" value={formatCurrency(sale.monto, sale.moneda)} />
        <Info label="Modalidad" value={PAYMENT_METHOD_LABELS[sale.modalidadPago] ?? sale.modalidadPago} />
        <Info label="Vendedor" value={`${clientLabel(sale.vendedor)}${sale.inmobiliaria ? ` (${sale.inmobiliaria.razonSocial})` : ''}`} />
        <Info label="Fecha de venta" value={formatDate(sale.fechaCreacion)} />
      </CardContent>
    </Card>
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
