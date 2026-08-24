import { ToggleGroup, ToggleGroupItem } from '../../../shared/ui/toggle-group'
import { DXF_LAYER_LABELS, DXF_LAYERS, type DxfLayer } from '../types'

type DxfLayerTogglesProps = {
  visibleLayers: ReadonlySet<DxfLayer>
  onVisibleLayersChange: (layers: ReadonlySet<DxfLayer>) => void
}

export default function DxfLayerToggles({
  visibleLayers,
  onVisibleLayersChange,
}: DxfLayerTogglesProps) {
  return (
    <ToggleGroup
      multiple
      variant="outline"
      className="w-full md:w-fit"
      value={[...visibleLayers]}
      onValueChange={(next) => {
        onVisibleLayersChange(new Set(next.filter(isDxfLayer)))
      }}
      aria-label="Capas del plano"
    >
      {DXF_LAYERS.map((layer) => (
        <ToggleGroupItem
          key={layer}
          value={layer}
          aria-label={DXF_LAYER_LABELS[layer]}
          className="min-h-11 min-w-11 flex-1 touch-manipulation md:min-h-8 md:flex-none"
        >
          {DXF_LAYER_LABELS[layer]}
        </ToggleGroupItem>
      ))}
    </ToggleGroup>
  )
}

function isDxfLayer(value: string): value is DxfLayer {
  return (DXF_LAYERS as readonly string[]).includes(value)
}
