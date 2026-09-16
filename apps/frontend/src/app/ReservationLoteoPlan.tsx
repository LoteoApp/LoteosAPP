import { useMemo } from 'react'
import { Alert, AlertDescription } from '../shared/ui/alert'
import LoteoPlanPanel from '../features/lots/components/LoteoPlanPanel'
import { useLayerVisibility } from '../features/lots/hooks/use-layer-visibility'
import { useLoteo } from '../features/lots/hooks/use-loteo'
import { planFromLoteoDetail, planLabelsFromLoteoDetail } from '../features/lots/lib/planFromLoteoDetail'
import type { LoteoDetail } from '../features/lots/types'
import ReservationDetailsPlanSkeleton from '../features/reservations/components/ReservationDetailsPlanSkeleton'
import type { Reservation } from '../features/reservations/types'

type ReservationLoteoPlanProps = {
  loteo: LoteoDetail
  selectedLoteId: string
  variant?: 'full' | 'reference'
}

export default function ReservationLoteoPlan({
  loteo,
  selectedLoteId,
  variant = 'full',
}: ReservationLoteoPlanProps) {
  const layers = useLayerVisibility()
  const selectedPolygonId = selectedLoteId ? `lote-${selectedLoteId}` : null
  const plan = useMemo(() => {
    const completePlan = planFromLoteoDetail(loteo)
    if (variant === 'full') return completePlan

    return completePlan.map((polygon) => (
      polygon.layer === 'LOTES' && polygon.id !== selectedPolygonId
        ? { ...polygon, caption: undefined, lotState: undefined }
        : polygon
    ))
  }, [loteo, selectedPolygonId, variant])
  const labels = useMemo(() => planLabelsFromLoteoDetail(loteo), [loteo])

  return (
    <LoteoPlanPanel
      className={variant === 'reference' ? 'h-72 min-w-0 sm:h-[28rem]' : 'min-w-0'}
      polygons={plan}
      visibleLayers={layers.visibleLayers}
      onVisibleLayersChange={layers.onVisibleLayersChange}
      selectedPolygonId={selectedPolygonId}
      polygonLabels={labels}
    />
  )
}

export function ReservationDetailsPlan({ accessToken, reservation }: { accessToken: string; reservation: Reservation }) {
  return <LoteReferencePlan accessToken={accessToken} loteoId={reservation.loteoId} loteId={reservation.loteId} />
}

// The reference plan of one lote, loaded by loteo id: what a reserva or a
// venta detail shows next to its data.
export function LoteReferencePlan({ accessToken, loteoId, loteId }: { accessToken: string; loteoId: string; loteId: string }) {
  const state = useLoteo(loteoId, accessToken)
  if (state.status === 'loading') return <ReservationDetailsPlanSkeleton message="Cargando plano del loteo…" />
  if (state.status !== 'loaded') {
    return <Alert variant="destructive"><AlertDescription>No se pudo cargar el plano del loteo.</AlertDescription></Alert>
  }
  return <ReservationLoteoPlan loteo={state.loteo} selectedLoteId={loteId} variant="reference" />
}
