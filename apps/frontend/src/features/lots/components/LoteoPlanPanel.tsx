import { useMemo } from 'react'
import { Info } from 'lucide-react'
import { cn } from '../../../shared/lib/utils'
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '../../../shared/ui/card'
import { LOT_STATES, type DxfLayer, type DxfPolygon, type LotState } from '../types'
import DxfLayerToggles from './DxfLayerToggles'
import DxfViewer from './DxfViewer'
import LotStateLegend from './LotStateLegend'

const NO_PLAN_MESSAGE = 'Este loteo todavía no tiene un plano cargado.'

type LoteoPlanPanelProps = {
  polygons: DxfPolygon[]
  visibleLayers: ReadonlySet<DxfLayer>
  onVisibleLayersChange: (layers: ReadonlySet<DxfLayer>) => void
  className?: string
  selectedPolygonId?: string | null
  onSelectPolygon?: (polygonId: string | null) => void
  polygonLabels?: ReadonlyMap<string, string>
}

export default function LoteoPlanPanel({
  polygons,
  visibleLayers,
  onVisibleLayersChange,
  className,
  selectedPolygonId,
  onSelectPolygon,
  polygonLabels,
}: LoteoPlanPanelProps) {
  const hasPlan = polygons.length > 0
  const counts = useMemo(() => countByState(polygons), [polygons])
  const legendStates = useMemo(
    () => LOT_STATES.filter((state) => counts[state] > 0),
    [counts],
  )
  const showLegend = visibleLayers.has('LOTES') && legendStates.length > 0
  const lotsCoverManzanas =
    onSelectPolygon !== undefined &&
    visibleLayers.has('LOTES') &&
    visibleLayers.has('MANZANA')

  return (
    <Card size="sm" className={cn('flex min-h-0 flex-col', className)}>
      <CardHeader>
        <CardTitle>Plano</CardTitle>
        {!hasPlan && (
          <CardDescription>Se ve acá cuando el agrimensor suba el DXF.</CardDescription>
        )}
        {showLegend && (
          <CardAction>
            <LotStateLegend states={legendStates} counts={counts} />
          </CardAction>
        )}
      </CardHeader>
      <CardContent className="flex min-h-0 flex-1 flex-col gap-2">
        {hasPlan && (
          <DxfLayerToggles
            visibleLayers={visibleLayers}
            onVisibleLayersChange={onVisibleLayersChange}
          />
        )}
        {lotsCoverManzanas ? (
          <p className="flex items-start gap-1.5 text-xs text-muted-foreground">
            <Info aria-hidden className="mt-px size-3.5 shrink-0" />
            Los lotes se dibujan encima de las manzanas. Para seleccionar una
            manzana, apagá la capa Lotes.
          </p>
        ) : null}
        <DxfViewer
          polygons={polygons}
          visibleLayers={visibleLayers}
          emptyMessage={NO_PLAN_MESSAGE}
          selectedPolygonId={selectedPolygonId}
          onSelectPolygon={onSelectPolygon}
          polygonLabels={polygonLabels}
        />
      </CardContent>
    </Card>
  )
}

function countByState(polygons: DxfPolygon[]): Record<LotState, number> {
  const counts: Record<LotState, number> = {
    disponible: 0,
    reservado: 0,
    vendido: 0,
    finalizado: 0,
  }
  for (const polygon of polygons) {
    if (polygon.layer === 'LOTES' && polygon.lotState) {
      counts[polygon.lotState] += 1
    }
  }
  return counts
}
