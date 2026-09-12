import { useMemo } from 'react'
import { Alert, AlertDescription } from '../shared/ui/alert'
import LoteoPlanPanel from '../features/lots/components/LoteoPlanPanel'
import { useLayerVisibility } from '../features/lots/hooks/use-layer-visibility'
import { useLoteo } from '../features/lots/hooks/use-loteo'
import { planFromLoteoDetail, planLabelsFromLoteoDetail } from '../features/lots/lib/planFromLoteoDetail'
import type { LoteoDetail } from '../features/lots/types'
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
      className={variant === 'reference' ? 'h-[28rem] min-w-0' : 'min-w-0'}
      polygons={plan}
      visibleLayers={layers.visibleLayers}
      onVisibleLayersChange={layers.onVisibleLayersChange}
      selectedPolygonId={selectedPolygonId}
      polygonLabels={labels}
    />
  )
}

export function ReservationDetailsPlan({ accessToken, reservation }: { accessToken: string; reservation: Reservation }) {
  const state = useLoteo(reservation.loteoId, accessToken)
  if (state.status === 'loading') return <p className="text-sm text-muted-foreground">Cargando plano del loteo…</p>
  if (state.status !== 'loaded') {
    return <Alert variant="destructive"><AlertDescription>No se pudo cargar el plano del loteo.</AlertDescription></Alert>
  }
  return <ReservationLoteoPlan loteo={state.loteo} selectedLoteId={reservation.loteId} />
}
