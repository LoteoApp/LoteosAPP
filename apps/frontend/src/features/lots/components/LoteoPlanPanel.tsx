import { Info } from 'lucide-react'
import { cn } from '../../../shared/lib/utils'
import { Alert, AlertDescription } from '../../../shared/ui/alert'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '../../../shared/ui/card'
import type { DxfLayer, DxfPolygon } from '../types'
import DxfLayerToggles from './DxfLayerToggles'
import DxfViewer from './DxfViewer'

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
  const lotsCoverManzanas =
    onSelectPolygon !== undefined &&
    visibleLayers.has('LOTES') &&
    visibleLayers.has('MANZANA')

  return (
    <Card size="sm" className={cn('flex min-h-0 flex-col', className)}>
      <CardHeader>
        <CardTitle>Plano</CardTitle>
        <CardDescription>
          {hasPlan
            ? 'Contorno, manzanas, lotes y calles del DXF cargado.'
            : 'Se ve acá cuando el agrimensor suba el DXF.'}
        </CardDescription>
      </CardHeader>
      <CardContent className="flex min-h-0 flex-1 flex-col gap-3">
        {hasPlan && (
          <DxfLayerToggles
            visibleLayers={visibleLayers}
            onVisibleLayersChange={onVisibleLayersChange}
          />
        )}
        {lotsCoverManzanas ? (
          <Alert>
            <Info aria-hidden />
            <AlertDescription>
              Los lotes se dibujan encima de las manzanas. Para seleccionar una
              manzana, apagá la capa Lotes.
            </AlertDescription>
          </Alert>
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
