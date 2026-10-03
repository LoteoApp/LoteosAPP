import { useRef, useState, type ReactNode } from 'react'
import { ArrowLeft, Printer } from 'lucide-react'
import { Link } from 'react-router'
import { ApiError } from '../../../shared/api/client'
import { newIdempotencyKey } from '../../../shared/lib/idempotencyKey'
import { formatCurrency } from '../../../shared/lib/formatCurrency'
import { formatDateTime } from '../../../shared/lib/formatDateTime'
import { Alert, AlertDescription, AlertTitle } from '../../../shared/ui/alert'
import { Button, buttonVariants } from '../../../shared/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card'
import SaleCreatePageSkeleton from '../components/SaleCreatePageSkeleton'
import SalePaymentForm from '../components/SalePaymentForm'
import SaleReceiptDialog from '../components/SaleReceiptDialog'
import {
  clientOptionLabel,
  lotOptionLabel,
  reservationSaleBlockedReason,
  saleReceiptFromSale,
  sellerAgencyLabel,
  sellerOptionLabel,
} from '../types'
import type { ReservationSaleContext, Sale, SalePaymentTerms } from '../types'

// One mounted page is one conversion attempt; the caller remounts it (a
// `key`) for another reserva or user.
export type ReservationSalePageProps = {
  reservationId: string
  context: ReservationSaleContext | null
  status: 'loading' | 'loaded' | 'not-found' | 'error'
  error?: string
  // A refresh that failed while the last loaded data is still shown.
  refreshError?: string
  convert: (terms: SalePaymentTerms, idempotencyKey: string) => Promise<Sale>
  onRefresh?: () => void
  renderPlan?: ReactNode
}

const UNCERTAIN_MESSAGE =
  'No pudimos confirmar si la venta se registró. Reintentá con las mismas condiciones: si ya se registró, vas a ver la venta.'

// A request that never got an answer, timed out or got a server failure may
// still have committed; any other 4xx proves nothing was written.
function isUncertain(error: unknown): boolean {
  return !(error instanceof ApiError) || error.status === 0 || error.status === 408 || error.status >= 500
}

export default function ReservationSalePage({
  reservationId,
  context,
  status,
  error,
  refreshError,
  convert,
  onRefresh,
  renderPlan,
}: ReservationSalePageProps) {
  const [createdSale, setCreatedSale] = useState<Sale | null>(null)
  const [isReceiptOpen, setIsReceiptOpen] = useState(false)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState<string | null>(null)
  const [canRefresh, setCanRefresh] = useState(false)
  // One key per conversion attempt: a retry reuses it so the backend hands
  // back the venta if the first request did land.
  const [idempotencyKey, setIdempotencyKey] = useState(newIdempotencyKey)
  // The terms of an attempt whose outcome is unknown: every retry sends these
  // under the same key, whatever the form shows by then.
  const [uncertainTerms, setUncertainTerms] = useState<SalePaymentTerms | null>(null)
  const inFlight = useRef(false)

  async function handleSubmit(terms: SalePaymentTerms): Promise<boolean> {
    if (inFlight.current) return false
    inFlight.current = true
    const sentTerms = uncertainTerms ?? terms
    setIsSubmitting(true)
    setSubmitError(null)
    setCanRefresh(false)
    try {
      const sale = await convert(sentTerms, idempotencyKey)
      setCreatedSale(sale)
      setIsReceiptOpen(true)
      setIdempotencyKey(newIdempotencyKey())
      setUncertainTerms(null)
      onRefresh?.()
      return true
    } catch (convertError) {
      if (isUncertain(convertError)) {
        setUncertainTerms(sentTerms)
        setSubmitError(UNCERTAIN_MESSAGE)
      } else {
        setUncertainTerms(null)
        setSubmitError((convertError as ApiError).message)
        setCanRefresh(onRefresh !== undefined)
      }
      return false
    } finally {
      inFlight.current = false
      setIsSubmitting(false)
    }
  }

  if (createdSale !== null) {
    return (
      <section className="flex min-h-0 flex-1 flex-col gap-4">
        <BackLink reservationId={reservationId} />
        <PageHeader />
        <ConvertedSale sale={createdSale} reservationId={reservationId} onPrint={() => setIsReceiptOpen(true)} />
        <SaleReceiptDialog
          open={isReceiptOpen}
          receipt={saleReceiptFromSale(createdSale)}
          onClose={() => setIsReceiptOpen(false)}
        />
      </section>
    )
  }

  if (status === 'loading') {
    return (
      <section className="flex min-h-0 flex-1 flex-col gap-4">
        <BackLink reservationId={reservationId} />
        <PageHeader />
        <SaleCreatePageSkeleton />
      </section>
    )
  }

  if (status !== 'loaded' || context === null) {
    return (
      <section className="flex flex-col gap-4">
        <BackLink reservationId={reservationId} />
        <Alert variant="destructive">
          <AlertTitle>No se puede convertir la reserva</AlertTitle>
          <AlertDescription>
            {status === 'not-found' ? 'No encontramos esta reserva.' : error ?? 'No se pudo cargar la reserva.'}
          </AlertDescription>
        </Alert>
      </section>
    )
  }

  const blockedReason = reservationSaleBlockedReason(context)

  return (
    <section className="flex min-h-0 flex-1 flex-col gap-4">
      <BackLink reservationId={reservationId} />
      <PageHeader />
      <div className="grid min-w-0 gap-4 lg:grid-cols-12 lg:items-start">
        <div className="flex min-w-0 flex-col gap-4 lg:col-span-5">{renderPlan}</div>
        <div className="flex min-w-0 flex-col gap-4 lg:col-span-7">
          {refreshError && (
            <Alert variant="destructive">
              <AlertTitle>No se pudo actualizar la reserva</AlertTitle>
              <AlertDescription>{refreshError} Se muestran los últimos datos cargados.</AlertDescription>
            </Alert>
          )}
          {blockedReason !== null && (
            <Alert variant={context.saleId === undefined ? 'destructive' : 'default'}>
              <AlertTitle>No se puede convertir</AlertTitle>
              <AlertDescription>
                <p>{blockedReason}</p>
                {context.saleId !== undefined && (
                  <Link className={buttonVariants({ variant: 'outline', className: 'mt-2 w-fit' })} to={`/ventas/${context.saleId}`}>
                    Ver venta
                  </Link>
                )}
              </AlertDescription>
            </Alert>
          )}
          <SalePaymentForm
            key={reservationId}
            lot={context.lot}
            client={context.client}
            seller={context.seller}
            participants={<FixedParticipants context={context} />}
            onSubmit={handleSubmit}
            isSubmitting={isSubmitting}
            disabled={blockedReason !== null}
            termsLocked={uncertainTerms !== null}
            error={submitError && (
              <span className="flex flex-col gap-2">
                <span>{submitError}</span>
                {canRefresh && (
                  <Button type="button" variant="outline" className="w-fit" onClick={onRefresh}>
                    Actualizar reserva
                  </Button>
                )}
              </span>
            )}
          />
        </div>
      </div>
    </section>
  )
}

function FixedParticipants({ context }: { context: ReservationSaleContext }) {
  return (
    <dl aria-label="Datos de la reserva" className="grid gap-3 text-sm sm:grid-cols-2">
      <Info label="Lote" value={lotOptionLabel(context.lot)} />
      <Info label="Cliente" value={clientOptionLabel(context.client)} />
      <Info label="Vendedor" value={`${sellerOptionLabel(context.seller)} · ${sellerAgencyLabel(context.seller)}`} />
      <Info label="La reserva vence" value={formatDateTime(context.dueAt)} />
    </dl>
  )
}

function ConvertedSale({ sale, reservationId, onPrint }: { sale: Sale; reservationId: string; onPrint: () => void }) {
  const outcome = sale.estado === 'completada'
    ? 'Se pagó al contado: la venta quedó completada y el lote finalizado.'
    : 'La venta quedó activa y el lote vendido.'
  return (
    <Card>
      <CardHeader>
        <CardTitle>Venta registrada</CardTitle>
        <CardDescription>
          La reserva se convirtió en una venta a {sale.cliente.apellido}, {sale.cliente.nombre} por{' '}
          {formatCurrency(sale.monto, sale.moneda)}. {outcome}
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center">
        <Button onClick={onPrint}>
          <Printer aria-hidden />
          Imprimir recibo
        </Button>
        <Link className={buttonVariants({ variant: 'outline' })} to={`/ventas/${sale.id}`}>
          Ver venta
        </Link>
        <Link className={buttonVariants({ variant: 'ghost' })} to={`/reservas/${reservationId}`}>
          Volver a la reserva
        </Link>
      </CardContent>
    </Card>
  )
}

function BackLink({ reservationId }: { reservationId: string }) {
  return (
    <Link
      to={`/reservas/${reservationId}`}
      className="inline-flex w-fit items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
    >
      <ArrowLeft aria-hidden className="size-4" />
      Volver a la reserva
    </Link>
  )
}

function PageHeader() {
  return (
    <div>
      <h1 className="text-2xl font-semibold text-foreground">Convertir reserva en venta</h1>
      <p className="text-sm text-muted-foreground">
        El lote, el cliente y el vendedor son los de la reserva. Elegí las condiciones de pago.
      </p>
    </div>
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
