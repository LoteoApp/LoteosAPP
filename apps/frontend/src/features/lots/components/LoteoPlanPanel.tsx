import { cn } from '../../../shared/lib/utils'
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
}

export default function LoteoPlanPanel({
  polygons,
  visibleLayers,
  onVisibleLayersChange,
  className,
}: LoteoPlanPanelProps) {
  const hasPlan = polygons.length > 0

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
        <DxfViewer
          polygons={polygons}
          visibleLayers={visibleLayers}
          emptyMessage={NO_PLAN_MESSAGE}
        />
      </CardContent>
    </Card>
  )
}
