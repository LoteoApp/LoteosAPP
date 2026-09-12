import { useState, type ReactNode } from 'react'
import { ArrowLeft, Download } from 'lucide-react'
import { Link } from 'react-router'
import { Alert, AlertDescription, AlertTitle } from '../../../shared/ui/alert'
import { Button, buttonVariants } from '../../../shared/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card'
import ReservationCreatePlanSkeleton from '../components/ReservationCreatePlanSkeleton'
import ReservationForm from '../components/ReservationForm'
import { downloadReservationReceipt } from '../api/reservations'
import { useEligibleSellers } from '../hooks/use-eligible-sellers'
import { useReservationMutations } from '../hooks/use-reservation-mutations'
import type { Reservation, ReservationClient, ReservationCreateDevelopment } from '../types'

type ReservationCreatePageProps = {
  accessToken?: string
  loteoId: string
  loteId: string
  loteo: ReservationCreateDevelopment | null
  loteoStatus: 'loading' | 'loaded' | 'not-found' | 'error'
  loteoError?: string
  clients: ReservationClient[]
  clientsLoading: boolean
  clientsError: string | null
  isAgencyUser: boolean
  renderPlan?: ReactNode
  renderClientAction?: ReactNode
  renderClientDialog?: ReactNode
}

export default function ReservationCreatePage({
  accessToken = '',
  loteoId,
  loteId,
  loteo,
  loteoStatus,
  loteoError,
  clients,
  clientsLoading,
  clientsError,
  isAgencyUser,
  renderPlan,
  renderClientAction,
  renderClientDialog,
}: ReservationCreatePageProps) {
  const token = accessToken
  const sellersState = useEligibleSellers(token, loteoId)
  const mutations = useReservationMutations(token)
  const [createdReservation, setCreatedReservation] = useState<Reservation | null>(null)
  const [receiptError, setReceiptError] = useState<string | null>(null)
  const [isDownloadingReceipt, setIsDownloadingReceipt] = useState(false)

  const selectedLot = loteo?.lotes.find((lot) => lot.id === loteId) ?? null
  const selectedBlock = loteo?.manzanas.find((block) => block.id === selectedLot?.manzanaId) ?? null
  const fixedSeller = isAgencyUser ? sellersState.sellers[0] : undefined
  const isUnavailable = selectedLot === null || selectedLot.estado !== 'disponible'

  async function handleSubmit(values: { loteoId: string; loteId: string; clienteId: string; vendedorId?: string }, key: string) {
    const created = await mutations.create(values, key)
    if (created) {
      setCreatedReservation(created)
      void handleDownloadReceipt(created)
    }
    return created !== null
  }

  async function handleDownloadReceipt(target = createdReservation) {
    if (!target) return
    setIsDownloadingReceipt(true)
    setReceiptError(null)
    try {
      const blob = await downloadReservationReceipt(token, target.id)
      const url = URL.createObjectURL(blob)
      const anchor = document.createElement('a')
      anchor.href = url
      anchor.download = `comprobante-reserva-${target.id}.pdf`
      anchor.click()
      window.setTimeout(() => URL.revokeObjectURL(url), 1000)
    } catch (downloadError) {
      setReceiptError(downloadError instanceof Error ? downloadError.message : 'No se pudo descargar el comprobante.')
    } finally {
      setIsDownloadingReceipt(false)
    }
  }

  if (loteoStatus === 'loading') {
    return (
      <section className="flex min-h-0 flex-1 flex-col gap-4">
        <BackLink loteoId={loteoId} />
        <ReservationCreateHeader />
        <div className="grid min-w-0 gap-4 lg:grid-cols-12 lg:items-start">
          <div className="min-w-0 lg:col-span-5">
            <ReservationCreatePlanSkeleton />
          </div>
        </div>
      </section>
    )
  }

  if (loteoStatus !== 'loaded' || loteo === null) {
    const message = loteoStatus === 'not-found'
      ? 'No encontramos este loteo.'
      : loteoStatus === 'error'
        ? loteoError
        : 'No se pudo cargar el loteo.'
    return <section className="flex flex-col gap-4"><BackLink loteoId={loteoId} /><Alert variant="destructive"><AlertTitle>No se puede iniciar la reserva</AlertTitle><AlertDescription>{message}</AlertDescription></Alert></section>
  }

  return (
    <section className="flex min-h-0 flex-1 flex-col gap-4">
      <BackLink loteoId={loteo.id} />
      <ReservationCreateHeader />
      <div className="grid min-w-0 gap-4 lg:grid-cols-12 lg:items-start">
        <div className="flex min-w-0 flex-col gap-4 lg:col-span-5">
          {renderPlan}
          <Card>
            <CardHeader>
              <CardTitle>{loteo.nombre}</CardTitle>
              <CardDescription>{loteo.ubicacion}{selectedBlock ? ` · Manzana ${selectedBlock.numero || 'sin número'}` : ''}{selectedLot ? ` · Lote ${selectedLot.numero || 'sin número'}` : ''}</CardDescription>
            </CardHeader>
            <CardContent className="grid gap-4 text-sm">
              {loteo.descripcion && <p className="text-muted-foreground">{loteo.descripcion}</p>}
              {selectedLot ? <dl className="grid gap-2 sm:grid-cols-3"><Info label="Estado" value={selectedLot.estado} /><Info label="Precio" value={selectedLot.precio === null ? 'A consultar' : `${selectedLot.moneda} ${selectedLot.precio.toLocaleString('es-AR')}`} /><Info label="Superficie" value={selectedLot.superficie === null ? 'Sin informar' : `${selectedLot.superficie} m²`} /></dl> : <Alert variant="destructive"><AlertDescription>El lote seleccionado no existe.</AlertDescription></Alert>}
              {selectedLot?.caracteristicas && <Info label="Características del lote" value={selectedLot.caracteristicas} />}
              {selectedBlock && <Info label="Servicios de la manzana" value={[
                selectedBlock.tieneAgua && 'Agua',
                selectedBlock.tieneCloaca && 'Cloaca',
                selectedBlock.tieneLuz && 'Luz',
                selectedBlock.tieneGas && 'Gas',
              ].filter(Boolean).join(' · ') || 'Sin servicios informados'} />}
              {isUnavailable && selectedLot && <Alert variant="destructive"><AlertTitle>Lote no disponible</AlertTitle><AlertDescription>El lote cambió de estado. Volvé al visor para elegir otro.</AlertDescription></Alert>}
            </CardContent>
          </Card>
        </div>
        <div className="flex min-w-0 flex-col gap-4 lg:col-span-7">
          {clientsError && <Alert variant="destructive"><AlertTitle>No se pudieron cargar los clientes</AlertTitle><AlertDescription>{clientsError}</AlertDescription></Alert>}
          {sellersState.error && <Alert variant="destructive"><AlertTitle>No se pudieron cargar los vendedores</AlertTitle><AlertDescription>{sellersState.error}</AlertDescription></Alert>}
          {createdReservation ? (
            <Card>
              <CardHeader><CardTitle>Reserva creada</CardTitle><CardDescription>La reserva quedó registrada y vence el {new Date(createdReservation.fechaVencimiento).toLocaleString('es-AR')}.</CardDescription></CardHeader>
              <CardContent className="flex flex-col gap-3 sm:flex-row sm:items-center">
                <Button onClick={() => void handleDownloadReceipt()} disabled={isDownloadingReceipt}><Download aria-hidden />{isDownloadingReceipt ? 'Generando comprobante…' : 'Descargar comprobante PDF'}</Button>
                <Link className={buttonVariants({ variant: 'outline' })} to={`/reservas/${createdReservation.id}`}>Ver detalle</Link>
              </CardContent>
              {receiptError && <Alert variant="destructive" className="mx-6 mb-6"><AlertDescription>{receiptError}</AlertDescription></Alert>}
            </Card>
          ) : (
            <ReservationForm
              loteos={[{ id: loteo.id, nombre: loteo.nombre }]}
              selectedLoteoId={loteo.id}
              selectedLoteId={loteId}
              lots={selectedLot && !isUnavailable ? [selectedLot] : []}
              clients={clients}
              sellers={sellersState.sellers}
              isLoadingSellers={sellersState.isLoading}
              isSubmitting={mutations.isSubmitting}
              error={mutations.error}
              fixedTarget
              disabled={isUnavailable || clientsLoading || sellersState.isLoading}
              sellerIsFixed={isAgencyUser}
              fixedSeller={fixedSeller}
              sellerRequired={!isAgencyUser}
              renderClientAction={renderClientAction}
              onSubmit={handleSubmit}
            />
          )}
        </div>
      </div>
      {renderClientDialog}
    </section>
  )
}

function BackLink({ loteoId }: { loteoId: string }) {
  return <Link to={`/lotes/${loteoId}`} className="inline-flex w-fit items-center gap-1 text-sm text-muted-foreground hover:text-foreground"><ArrowLeft aria-hidden className="size-4" />Volver al loteo</Link>
}

function ReservationCreateHeader() {
  return (
    <div>
      <h1 className="text-2xl font-semibold text-foreground">Nueva reserva</h1>
      <p className="text-sm text-muted-foreground">Corroborá el loteo y el lote antes de asociar el cliente.</p>
    </div>
  )
}

function Info({ label, value }: { label: string; value: string }) {
  return <div><dt className="text-xs font-medium text-muted-foreground">{label}</dt><dd className="capitalize">{value}</dd></div>
}
