import { cn } from '../../../shared/lib/utils'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '../../../shared/ui/card'
import DxfFileField from './DxfFileField'
import DxfLayerToggles from './DxfLayerToggles'
import DxfParseAlerts from './DxfParseAlerts'
import DxfPlanEmptyState from './DxfPlanEmptyState'
import DxfViewer from './DxfViewer'
import type { DxfLayer, DxfParseResult, DxfPolygon, DxfValidationIssue } from '../types'

type LoteoPlanCardProps = {
  hasPlan: boolean
  fileName: string | null
  error: string | null
  issues: DxfValidationIssue[]
  polygons: DxfPolygon[]
  visibleLayers: ReadonlySet<DxfLayer>
  onVisibleLayersChange: (layers: ReadonlySet<DxfLayer>) => void
  onParsed: (result: DxfParseResult, file: File) => void
  onError: (message: string) => void
  onCleared: () => void
  className?: string
}

export default function LoteoPlanCard({
  hasPlan,
  fileName,
  error,
  issues,
  polygons,
  visibleLayers,
  onVisibleLayersChange,
  onParsed,
  onError,
  onCleared,
  className,
}: LoteoPlanCardProps) {
  return (
    <Card size="sm" className={cn('flex min-h-0 flex-col', className)}>
      <CardHeader>
        <CardTitle>Plano</CardTitle>
        <CardDescription>
          {hasPlan
            ? `Cargado desde ${fileName ?? 'el DXF'}.`
            : 'Opcional. Lo carga quien tenga el DXF.'}
        </CardDescription>
      </CardHeader>
      <CardContent className="flex min-h-0 flex-1 flex-col gap-3">
        <DxfFileField
          fileName={fileName}
          onParsed={onParsed}
          onError={onError}
          onCleared={onCleared}
        />
        <DxfParseAlerts error={error} issues={issues} />
        {hasPlan ? (
          <>
            <DxfLayerToggles
              visibleLayers={visibleLayers}
              onVisibleLayersChange={onVisibleLayersChange}
            />
            <DxfViewer polygons={polygons} visibleLayers={visibleLayers} />
          </>
        ) : (
          <DxfPlanEmptyState />
        )}
      </CardContent>
    </Card>
  )
}
