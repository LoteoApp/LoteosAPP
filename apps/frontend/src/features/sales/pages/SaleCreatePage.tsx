import { useState, type ReactNode } from 'react'
import { ArrowLeft, Printer } from 'lucide-react'
import { Link } from 'react-router'
import { messageFromError } from '../../../shared/api/client'
import { newIdempotencyKey } from '../../../shared/lib/idempotencyKey'
import { Alert, AlertDescription, AlertTitle } from '../../../shared/ui/alert'
import { Button, buttonVariants } from '../../../shared/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card'
import { formatArea } from '../../../shared/lib/formatArea'
import { formatCurrency } from '../../../shared/lib/formatCurrency'
import SaleForm from '../components/SaleForm'
import SaleCreatePageSkeleton from '../components/SaleCreatePageSkeleton'
import SaleReceiptDialog from '../components/SaleReceiptDialog'
import { lotOptionFromDevelopment, saleReceiptFromSale } from '../types'
import type {
  ClientOption,
  CreateSaleValues,
  Sale,
  SaleCreateDevelopment,
  SellerOption,
} from '../types'

export type SaleCreatePageProps = {
  loteoId: string
  loteId: string
  loteo: SaleCreateDevelopment | null
  loteoStatus: 'loading' | 'loaded' | 'not-found' | 'error'
  loteoError?: string
  clients: readonly ClientOption[]
  clientsLoading?: boolean
  clientsError?: string | null
  createdClient?: ClientOption | null
  onRegisterClient?: () => void
  renderClientDialog?: ReactNode
  loadSellers: (loteoId: string, signal?: AbortSignal) => Promise<SellerOption[]>
  createSale: (values: CreateSaleValues, idempotencyKey: string) => Promise<Sale>
  renderPlan?: ReactNode
}

export default function SaleCreatePage({
  loteoId,
  loteId,
  loteo,
  loteoStatus,
  loteoError,
  clients,
  clientsLoading = false,
  clientsError = null,
  createdClient = null,
  onRegisterClient,
  renderClientDialog,
  loadSellers,
  createSale,
  renderPlan,
}: SaleCreatePageProps) {
  const [createdSale, setCreatedSale] = useState<Sale | null>(null)
  const [isReceiptOpen, setIsReceiptOpen] = useState(false)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState<string | null>(null)
  // One key per attempted sale: a retry after a failure reuses it so the
  // backend can hand back the venta if the first request did land.
  const [idempotencyKey, setIdempotencyKey] = useState(newIdempotencyKey)

  async function handleSubmit(values: CreateSaleValues): Promise<boolean> {
    setIsSubmitting(true)
    setSubmitError(null)
    try {
      const sale = await createSale(values, idempotencyKey)
      setIdempotencyKey(newIdempotencyKey())
      setCreatedSale(sale)
      setIsReceiptOpen(true)
      return true
    } catch (error) {
      setSubmitError(messageFromError(error))
      return false
    } finally {
      setIsSubmitting(false)
    }
  }

  if (loteoStatus === 'loading') {
    return (
      <section className="flex min-h-0 flex-1 flex-col gap-4">
        <BackLink loteoId={loteoId} />
        <SaleCreateHeader />
        <SaleCreatePageSkeleton />
      </section>
    )
  }

  if (loteoStatus !== 'loaded' || loteo === null) {
    const message = loteoStatus === 'not-found'
      ? 'No encontramos este loteo.'
      : loteoStatus === 'error'
        ? loteoError
        : 'No se pudo cargar el loteo.'
    return (
      <section className="flex flex-col gap-4">
        <BackLink loteoId={loteoId} />
        <Alert variant="destructive">
          <AlertTitle>No se puede iniciar la venta</AlertTitle>
          <AlertDescription>{message}</AlertDescription>
        </Alert>
      </section>
    )
  }

  const selectedLot = loteo.lotes.find((lot) => lot.id === loteId) ?? null
  const selectedBlock = loteo.manzanas.find((block) => block.id === selectedLot?.manzanaId) ?? null
  const isUnavailable = selectedLot === null || selectedLot.estado !== 'disponible'
  const lot = isUnavailable ? null : lotOptionFromDevelopment(loteo, loteId)

  return (
    <section className="flex min-h-0 flex-1 flex-col gap-4">
      <BackLink loteoId={loteo.id} />
      <SaleCreateHeader />
      <div className="grid min-w-0 gap-4 lg:grid-cols-12 lg:items-start">
        <div className="flex min-w-0 flex-col gap-4 lg:col-span-5">
          {renderPlan}
          <Card>
            <CardHeader>
              <CardTitle>{loteo.nombre}</CardTitle>
              <CardDescription>
                {loteo.ubicacion}
                {selectedBlock ? ` · Manzana ${selectedBlock.numero || 'sin número'}` : ''}
                {selectedLot ? ` · Lote ${selectedLot.numero || 'sin número'}` : ''}
              </CardDescription>
            </CardHeader>
            <CardContent className="grid gap-4 text-sm">
              {loteo.descripcion && <p className="text-muted-foreground">{loteo.descripcion}</p>}
              {selectedLot ? (
                <dl className="grid gap-2 sm:grid-cols-3">
                  <Info label="Estado" value={createdSale ? 'vendido' : selectedLot.estado} className="capitalize" />
                  <Info
                    label="Precio"
                    value={selectedLot.precio === null ? 'A consultar' : formatCurrency(selectedLot.precio, selectedLot.moneda)}
                  />
                  <Info
                    label="Superficie"
                    value={selectedLot.superficie === null ? 'Sin informar' : formatArea(selectedLot.superficie)}
                  />
                </dl>
              ) : (
                <Alert variant="destructive">
                  <AlertDescription>El lote seleccionado no existe.</AlertDescription>
                </Alert>
              )}
              {selectedLot?.caracteristicas && (
                <Info label="Características del lote" value={selectedLot.caracteristicas} />
              )}
              {selectedBlock && (
                <Info
                  label="Servicios de la manzana"
                  value={[
                    selectedBlock.tieneAgua && 'Agua',
                    selectedBlock.tieneCloaca && 'Cloaca',
                    selectedBlock.tieneLuz && 'Luz',
                    selectedBlock.tieneGas && 'Gas',
                  ].filter(Boolean).join(' · ') || 'Sin servicios informados'}
                />
              )}
              {isUnavailable && selectedLot && !createdSale && (
                <Alert variant="destructive">
                  <AlertTitle>Lote no disponible</AlertTitle>
                  <AlertDescription>El lote cambió de estado. Volvé al visor para elegir otro.</AlertDescription>
                </Alert>
              )}
            </CardContent>
          </Card>
        </div>
        <div className="flex min-w-0 flex-col gap-4 lg:col-span-7">
          {createdSale ? (
            <Card>
              <CardHeader>
                <CardTitle>Venta registrada</CardTitle>
                <CardDescription>
                  El lote quedó vendido a {createdSale.cliente.apellido}, {createdSale.cliente.nombre} por{' '}
                  {formatCurrency(createdSale.monto, createdSale.moneda)}.
                </CardDescription>
              </CardHeader>
              <CardContent className="flex flex-col gap-3 sm:flex-row sm:items-center">
                <Button onClick={() => setIsReceiptOpen(true)}>
                  <Printer aria-hidden />
                  Imprimir recibo
                </Button>
                <Link className={buttonVariants({ variant: 'outline' })} to={`/ventas/${createdSale.id}`}>
                  Ver detalle
                </Link>
                <Link className={buttonVariants({ variant: 'ghost' })} to="/ventas">
                  Ir al listado
                </Link>
              </CardContent>
            </Card>
          ) : (
            <SaleForm
              lot={lot}
              clients={clients}
              clientsLoading={clientsLoading}
              clientsError={clientsError}
              createdClient={createdClient}
              onRegisterClient={onRegisterClient}
              renderClientDialog={renderClientDialog}
              loadSellers={loadSellers}
              onSubmit={handleSubmit}
              isSubmitting={isSubmitting}
              error={submitError}
              disabled={isUnavailable}
            />
          )}
        </div>
      </div>
      <SaleReceiptDialog
        open={isReceiptOpen && createdSale !== null}
        receipt={createdSale ? saleReceiptFromSale(createdSale) : null}
        onClose={() => setIsReceiptOpen(false)}
      />
    </section>
  )
}

function BackLink({ loteoId }: { loteoId: string }) {
  return (
    <Link
      to={`/lotes/${loteoId}`}
      className="inline-flex w-fit items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
    >
      <ArrowLeft aria-hidden className="size-4" />
      Volver al loteo
    </Link>
  )
}

function SaleCreateHeader() {
  return (
    <div>
      <h1 className="text-2xl font-semibold text-foreground">Nueva venta</h1>
      <p className="text-sm text-muted-foreground">
        Corroborá el lote y elegí el cliente comprador y quién realizó la venta.
      </p>
    </div>
  )
}

function Info({ label, value, className }: { label: string; value: string; className?: string }) {
  return (
    <div>
      <dt className="text-xs font-medium text-muted-foreground">{label}</dt>
      <dd className={className}>{value}</dd>
    </div>
  )
}
