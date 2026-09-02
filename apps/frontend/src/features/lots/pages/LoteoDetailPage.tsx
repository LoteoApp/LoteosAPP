import { useMemo, useState } from 'react'
import { ArrowLeft } from 'lucide-react'
import { Link, useParams } from 'react-router'
import { Alert, AlertDescription, AlertTitle } from '../../../shared/ui/alert'
import LoteoDetailHeader from '../components/LoteoDetailHeader'
import LoteoPlanPanel from '../components/LoteoPlanPanel'
import LotesTable from '../components/LotesTable'
import ManzanaFilter, { ALL_MANZANAS } from '../components/ManzanaFilter'
import { useLayerVisibility } from '../hooks/use-layer-visibility'
import { useLoteo } from '../hooks/use-loteo'
import { planFromLoteoDetail } from '../lib/planFromLoteoDetail'

type LoteoDetailPageProps = {
  accessToken: string | null
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

export default function LoteoDetailPage({ accessToken }: LoteoDetailPageProps) {
  const { loteoId = '' } = useParams()
  const state = useLoteo(loteoId, accessToken ?? '')
  const layers = useLayerVisibility()
  const [manzanaFilter, setManzanaFilter] = useState(ALL_MANZANAS)

  const loteo = state.status === 'loaded' ? state.loteo : null

  const plan = useMemo(() => (loteo ? planFromLoteoDetail(loteo) : []), [loteo])
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

      <div className="grid min-h-0 flex-1 grid-cols-1 gap-4 md:grid-cols-2">
        <LoteoPlanPanel
          className="md:sticky md:top-4 md:h-[calc(100dvh-7rem)] md:self-start"
          polygons={plan}
          visibleLayers={layers.visibleLayers}
          onVisibleLayersChange={layers.onVisibleLayersChange}
        />

        <div className="flex min-h-0 flex-col gap-3">
          <ManzanaFilter
            manzanas={state.loteo.manzanas}
            value={manzanaFilter}
            onChange={setManzanaFilter}
          />
          <LotesTable lotes={filteredLotes} manzanaNumberById={manzanaNumberById} />
        </div>
      </div>
    </section>
  )
}
