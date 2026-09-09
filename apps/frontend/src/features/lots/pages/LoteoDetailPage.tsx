import { useMemo, useState, type ReactNode } from 'react'
import { ArrowLeft } from 'lucide-react'
import { Link, useParams } from 'react-router'
import { Alert, AlertDescription, AlertTitle } from '../../../shared/ui/alert'
import { SaveNotice, useSaveNotice } from '../../../shared/ui/save-notice'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../../../shared/ui/tabs'
import LoteoDetailHeader from '../components/LoteoDetailHeader'
import LoteoPlanPanel from '../components/LoteoPlanPanel'
import LoteoSummaryPanel from '../components/LoteoSummaryPanel'
import LotesList from '../components/LotesList'
import LotesToolbar from '../components/LotesToolbar'
import ManzanaFilter, { ALL_MANZANAS } from '../components/ManzanaFilter'
import PlanSelectionPanel from '../components/PlanSelectionPanel'
import { useLayerVisibility } from '../hooks/use-layer-visibility'
import { useLoteo } from '../hooks/use-loteo'
import { usePlanSelection } from '../hooks/use-plan-selection'
import { useUpdateCalle } from '../hooks/use-update-calle'
import { useUpdateLote } from '../hooks/use-update-lote'
import { useUpdateManzana } from '../hooks/use-update-manzana'
import { planFromLoteoDetail, planLabelsFromLoteoDetail } from '../lib/planFromLoteoDetail'
import { countLotesByState } from '../lib/lotStateVisuals'
import type { LoteoDetail, LotState } from '../types'

const SUMMARY_TAB = 'resumen'
const LOTS_TAB = 'lotes'
const RESERVATIONS_TAB = 'reservas'

type LoteoDetailPageProps = {
  accessToken: string | null
  canEdit?: boolean
  renderReservationAction?: (lote: LoteoDetail['lotes'][number], onCreated: () => void) => ReactNode
  renderReservations?: (
    loteo: LoteoDetail,
    onReservationCanceled: (loteId: string) => void,
  ) => ReactNode
  renderReservationSummary?: (
    lote: LoteoDetail['lotes'][number],
    onReservationCanceled: () => void,
  ) => ReactNode
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
  renderReservations,
  renderReservationSummary,
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
  const [search, setSearch] = useState('')
  const [manzanaFilter, setManzanaFilter] = useState(ALL_MANZANAS)
  const [stateFilter, setStateFilter] = useState<ReadonlySet<LotState>>(new Set())
  const [tab, setTab] = useState(SUMMARY_TAB)
  const reservationNotice = useSaveNotice()
  const reservationCancelNotice = useSaveNotice()

  // React Router keeps this component mounted across a param change, so drop the
  // previous loteo's manzana filter and layer selection when loteoId changes.
  const [trackedLoteoId, setTrackedLoteoId] = useState(loteoId)
  if (loteoId !== trackedLoteoId) {
    setTrackedLoteoId(loteoId)
    setSearch('')
    setManzanaFilter(ALL_MANZANAS)
    setStateFilter(new Set())
    setTab(SUMMARY_TAB)
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
    if (selectedKey !== '' && selectedKey !== 'loteo') {
      setTab(LOTS_TAB)
    }
    loteUpdate.reset()
    manzanaUpdate.reset()
    calleUpdate.reset()
  }
  const manzanaNumberById = useMemo(
    () => new Map((loteo?.manzanas ?? []).map((manzana) => [manzana.id, manzana.numero])),
    [loteo],
  )
  const stateCounts = useMemo(() => countLotesByState(loteo?.lotes ?? []), [loteo])
  const filteredLotes = useMemo(
    () => filterLotes(loteo?.lotes ?? [], search, stateFilter, manzanaFilter, manzanaNumberById),
    [loteo, search, stateFilter, manzanaFilter, manzanaNumberById],
  )

  function handleReservationCreated(lote: LoteoDetail['lotes'][number]) {
    replaceLote({ ...lote, estado: 'reservado' })
    reservationNotice.show()
  }

  function handleReservationCanceled(lote: LoteoDetail['lotes'][number]) {
    replaceLote({ ...lote, estado: 'disponible' })
    reservationCancelNotice.show()
  }

  function handleReservationCanceledForLote(loteId: string) {
    const lote = loteo?.lotes.find((item) => item.id === loteId)
    if (lote) {
      handleReservationCanceled(lote)
    }
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

      <div className="grid min-h-0 min-w-0 flex-1 grid-cols-1 gap-4 lg:grid-cols-2">
        <LoteoPlanPanel
          className="min-w-0 lg:sticky lg:top-4 lg:h-[calc(100dvh-15rem)] lg:self-start"
          polygons={plan}
          visibleLayers={layers.visibleLayers}
          onVisibleLayersChange={layers.onVisibleLayersChange}
          selectedPolygonId={selection.selectedPolygonId}
          onSelectPolygon={selection.select}
          polygonLabels={polygonLabels}
        />

        <Tabs
          value={tab}
          onValueChange={(next) => setTab(String(next))}
          className="flex min-h-0 min-w-0 flex-col gap-3 lg:sticky lg:top-4 lg:h-[calc(100dvh-15rem)] lg:self-start"
        >
          <TabsList variant="line" className="h-auto! w-full justify-start gap-4 px-0">
            <TabsTrigger value={SUMMARY_TAB} className="min-h-11 flex-none md:min-h-9">
              Resumen
            </TabsTrigger>
            <TabsTrigger value={LOTS_TAB} className="min-h-11 flex-none md:min-h-9">
              Lotes
              <TabCount value={state.loteo.lotes.length} />
            </TabsTrigger>
            {renderReservations && (
              <TabsTrigger value={RESERVATIONS_TAB} className="min-h-11 flex-none md:min-h-9">
                Reservas
              </TabsTrigger>
            )}
          </TabsList>

          <TabsContent value={SUMMARY_TAB} className="min-h-0 overflow-y-auto">
            <LoteoSummaryPanel loteo={state.loteo} />
          </TabsContent>

          <TabsContent value={LOTS_TAB} className="flex min-h-0 flex-col gap-3">
            <ManzanaFilter
              manzanas={state.loteo.manzanas}
              value={manzanaFilter}
              onChange={setManzanaFilter}
            />
            <LotesToolbar
              search={search}
              onSearchChange={setSearch}
              states={stateFilter}
              onStatesChange={setStateFilter}
              counts={stateCounts}
            />

            <div className="min-h-0 flex-1 overflow-y-auto rounded-lg border border-border p-1">
              <LotesList
                lotes={filteredLotes}
                manzanaNumberById={manzanaNumberById}
                selectedLoteId={selection.selected?.kind === 'lote' ? selection.selected.id : null}
                onSelectLote={(loteId) => selection.selectEntity({ kind: 'lote', id: loteId })}
              />
            </div>

            <PlanSelectionPanel
              canEdit={canEdit}
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
              renderReservationSummary={
                renderReservationSummary
                  ? (lote) => renderReservationSummary(lote, () => handleReservationCanceled(lote))
                  : undefined
              }
            />
          </TabsContent>

          {renderReservations && (
            <TabsContent value={RESERVATIONS_TAB} className="min-h-0 overflow-y-auto">
              {renderReservations(state.loteo, handleReservationCanceledForLote)}
            </TabsContent>
          )}
        </Tabs>
      </div>
    </section>
  )
}

function TabCount({ value }: { value: number }) {
  return (
    <span
      aria-hidden
      className="rounded-full bg-muted px-1.5 py-0.5 text-xs font-medium tabular-nums text-muted-foreground"
    >
      {value}
    </span>
  )
}

function filterLotes(
  lotes: LoteoDetail['lotes'],
  search: string,
  states: ReadonlySet<LotState>,
  manzanaId: string,
  manzanaNumberById: ReadonlyMap<string, string>,
): LoteoDetail['lotes'] {
  const term = search.trim().toLocaleLowerCase()
  return lotes.filter((lote) => {
    if (manzanaId !== ALL_MANZANAS && lote.manzanaId !== manzanaId) {
      return false
    }
    if (states.size > 0 && !states.has(lote.estado)) {
      return false
    }
    if (term === '') {
      return true
    }
    const manzana = manzanaNumberById.get(lote.manzanaId) ?? ''
    return (
      lote.numero.toLocaleLowerCase().includes(term) ||
      manzana.toLocaleLowerCase().includes(term)
    )
  })
}
