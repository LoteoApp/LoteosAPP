import { useMemo, useState, type ReactNode } from 'react'
import { ArrowLeft } from 'lucide-react'
import { Link, useParams } from 'react-router'
import { Alert, AlertDescription, AlertTitle } from '../../../shared/ui/alert'
import { Card, CardContent, CardHeader, CardTitle } from '../../../shared/ui/card'
import { SaveNotice, useSaveNotice } from '../../../shared/ui/save-notice'
import ArchivosSection from '../components/ArchivosSection'
import LoteoDetailHeader from '../components/LoteoDetailHeader'
import LoteoPlanPanel from '../components/LoteoPlanPanel'
import LotesTable from '../components/LotesTable'
import ManzanaFilter, { ALL_MANZANAS } from '../components/ManzanaFilter'
import PlanSelectionPanel from '../components/PlanSelectionPanel'
import { useLayerVisibility } from '../hooks/use-layer-visibility'
import { useLoteo } from '../hooks/use-loteo'
import { usePlanSelection } from '../hooks/use-plan-selection'
import { useUpdateCalle } from '../hooks/use-update-calle'
import { useUpdateLote } from '../hooks/use-update-lote'
import { useUpdateManzana } from '../hooks/use-update-manzana'
import { planFromLoteoDetail, planLabelsFromLoteoDetail } from '../lib/planFromLoteoDetail'
import type { LoteoDetail } from '../types'

type LoteoDetailPageProps = {
  accessToken: string | null
  canEdit?: boolean
  renderReservationAction?: (lote: LoteoDetail['lotes'][number], onCreated: () => void) => ReactNode
  renderReservationCancelAction?: (lote: LoteoDetail['lotes'][number], onCanceled: () => void) => ReactNode
}

function BackLink() {
  return (
    <Link
      to="/lotes"
      className="inline-flex w-fit items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
    >
      <ArrowLeft aria-hidden className="size-4" />
      Volver al listado
    </Link>
  )
}

export default function LoteoDetailPage({
  accessToken,
  canEdit = false,
  renderReservationAction,
  renderReservationCancelAction,
}: LoteoDetailPageProps) {
  const { loteoId = '' } = useParams()
  const { replaceLote, replaceManzana, replaceCalle, ...state } = useLoteo(
    loteoId,
    accessToken ?? '',
  )
  const layers = useLayerVisibility()
  const loteUpdate = useUpdateLote(accessToken)
  const manzanaUpdate = useUpdateManzana(accessToken)
  const calleUpdate = useUpdateCalle(accessToken)
  const [manzanaFilter, setManzanaFilter] = useState(ALL_MANZANAS)
  const reservationNotice = useSaveNotice()
  const reservationCancelNotice = useSaveNotice()

  // React Router keeps this component mounted across a param change, so drop the
  // previous loteo's manzana filter and layer selection when loteoId changes.
  const [trackedLoteoId, setTrackedLoteoId] = useState(loteoId)
  if (loteoId !== trackedLoteoId) {
    setTrackedLoteoId(loteoId)
    setManzanaFilter(ALL_MANZANAS)
    layers.reset()
    loteUpdate.reset()
    manzanaUpdate.reset()
    calleUpdate.reset()
  }

  const loteo = state.status === 'loaded' ? state.loteo : null

  const plan = useMemo(() => (loteo ? planFromLoteoDetail(loteo) : []), [loteo])
  const polygonLabels = useMemo(
    () => (loteo ? planLabelsFromLoteoDetail(loteo) : new Map<string, string>()),
    [loteo],
  )
  const selection = usePlanSelection(plan)
  const selectedKey =
    selection.selected === null
      ? ''
      : selection.selected.kind === 'loteo'
        ? 'loteo'
        : `${selection.selected.kind}:${selection.selected.id}`
  const [trackedSelection, setTrackedSelection] = useState(selectedKey)
  if (selectedKey !== trackedSelection) {
    setTrackedSelection(selectedKey)
    loteUpdate.reset()
    manzanaUpdate.reset()
    calleUpdate.reset()
  }
  const manzanaNumberById = useMemo(
    () => new Map((loteo?.manzanas ?? []).map((manzana) => [manzana.id, manzana.numero])),
    [loteo],
  )
  const filteredLotes = useMemo(() => {
    const lotes = loteo?.lotes ?? []
    return manzanaFilter === ALL_MANZANAS
      ? lotes
      : lotes.filter((lote) => lote.manzanaId === manzanaFilter)
  }, [loteo, manzanaFilter])

  function handleReservationCreated(lote: LoteoDetail['lotes'][number]) {
    replaceLote({ ...lote, estado: 'reservado' })
    reservationNotice.show()
  }

  function handleReservationCanceled(lote: LoteoDetail['lotes'][number]) {
    replaceLote({ ...lote, estado: 'disponible' })
    reservationCancelNotice.show()
  }

  if (state.status === 'loading') {
    return (
      <section className="flex flex-col gap-4">
        <BackLink />
        <p className="text-muted-foreground">Cargando el loteo…</p>
      </section>
    )
  }

  if (state.status === 'not-found') {
    return (
      <section className="flex flex-col gap-4">
        <BackLink />
        <div className="flex flex-col items-center justify-center gap-2 rounded-xl border border-dashed border-border bg-muted/30 p-6 text-center">
          <p className="text-sm font-medium">No encontramos este loteo</p>
          <p className="max-w-xs text-sm text-muted-foreground">
            Puede que lo hayan dado de baja o que el enlace esté mal.
          </p>
        </div>
      </section>
    )
  }

  if (state.status === 'error') {
    return (
      <section className="flex flex-col gap-4">
        <BackLink />
        <Alert variant="destructive">
          <AlertTitle>No se pudo cargar el loteo</AlertTitle>
          <AlertDescription>{state.message}</AlertDescription>
        </Alert>
      </section>
    )
  }

  return (
    <section className="flex min-h-0 flex-1 flex-col gap-4">
      <LoteoDetailHeader loteo={state.loteo} hasPlan={plan.length > 0} />
      <SaveNotice token={reservationNotice.token}>Reserva creada</SaveNotice>
      <SaveNotice token={reservationCancelNotice.token}>Reserva cancelada</SaveNotice>

      <Card size="sm">
        <CardHeader>
          <CardTitle>Fotos y documentos</CardTitle>
        </CardHeader>
        <CardContent>
          {/* Remounted per loteo: React Router keeps this page mounted across
              a param change, so a pending upload/delete from the previous
              loteo must never land on this one once it resolves. */}
          <ArchivosSection
            key={state.loteo.id}
            target={{ kind: 'loteo', loteoId: state.loteo.id }}
            accessToken={accessToken}
            canEdit={canEdit}
            showLabel={false}
          />
        </CardContent>
      </Card>

      <div className="grid min-h-0 min-w-0 flex-1 grid-cols-1 gap-4 lg:grid-cols-2">
        <LoteoPlanPanel
          className="min-w-0 lg:sticky lg:top-4 lg:h-[calc(100dvh-7rem)] lg:self-start"
          polygons={plan}
          visibleLayers={layers.visibleLayers}
          onVisibleLayersChange={layers.onVisibleLayersChange}
          selectedPolygonId={selection.selectedPolygonId}
          onSelectPolygon={selection.select}
          polygonLabels={polygonLabels}
        />

        <div className="flex min-h-0 min-w-0 flex-col gap-3">
          <PlanSelectionPanel
            canEdit={canEdit}
            accessToken={accessToken}
            selected={selection.selected}
            loteo={state.loteo}
            polygonLabels={polygonLabels}
            selectedPolygonId={selection.selectedPolygonId}
            updateState={loteUpdate}
            onSave={async (loteId, payload) => {
              const updated = await loteUpdate.update(state.loteo.id, loteId, payload)
              if (updated) {
                replaceLote(updated)
              }
              return updated !== null
            }}
            manzanaUpdateState={manzanaUpdate}
            onSaveManzana={async (manzanaId, payload) => {
              const updated = await manzanaUpdate.update(state.loteo.id, manzanaId, payload)
              if (updated) {
                replaceManzana(updated)
              }
              return updated !== null
            }}
            calleUpdateState={calleUpdate}
            onSaveCalle={async (calleId, payload) => {
              const updated = await calleUpdate.update(state.loteo.id, calleId, payload)
              if (updated) {
                replaceCalle(updated)
              }
              return updated !== null
            }}
            renderReservationAction={
              renderReservationAction
                ? (lote) => renderReservationAction(lote, () => handleReservationCreated(lote))
                : undefined
            }
            renderReservationCancelAction={
              renderReservationCancelAction
                ? (lote) => renderReservationCancelAction(lote, () => handleReservationCanceled(lote))
                : undefined
            }
          />
          <ManzanaFilter
            manzanas={state.loteo.manzanas}
            value={manzanaFilter}
            onChange={setManzanaFilter}
          />
          <LotesTable
            lotes={filteredLotes}
            manzanaNumberById={manzanaNumberById}
            selectedLoteId={selection.selected?.kind === 'lote' ? selection.selected.id : null}
            onSelectLote={(loteId) => selection.selectEntity({ kind: 'lote', id: loteId })}
          />
        </div>
      </div>
    </section>
  )
}
