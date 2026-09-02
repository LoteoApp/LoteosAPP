import { useCallback, useState } from 'react'
import { DXF_LAYERS, type DxfLayer } from '../types'

function allLayersVisible(): ReadonlySet<DxfLayer> {
  return new Set(DXF_LAYERS)
}

export type UseLayerVisibility = {
  visibleLayers: ReadonlySet<DxfLayer>
  onVisibleLayersChange: (layers: ReadonlySet<DxfLayer>) => void
  reset: () => void
}

// Shared by the loteo form (useDxfPlan) and the loteo detail screen so both
// start with every layer visible and toggle the same way.
export function useLayerVisibility(): UseLayerVisibility {
  const [visibleLayers, setVisibleLayers] =
    useState<ReadonlySet<DxfLayer>>(allLayersVisible)

  const reset = useCallback(() => {
    setVisibleLayers(allLayersVisible())
  }, [])

  return { visibleLayers, onVisibleLayersChange: setVisibleLayers, reset }
}
