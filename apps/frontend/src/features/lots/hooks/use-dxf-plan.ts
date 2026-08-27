import { useCallback, useState } from 'react'
import { DXF_LAYERS, type DxfLayer, type DxfParseResult, type DxfValidationIssue } from '../types'

type DxfPlanState =
  | { status: 'empty' }
  | { status: 'loaded'; result: DxfParseResult; fileName: string }
  | { status: 'error'; message: string }

const EMPTY_STATE: DxfPlanState = { status: 'empty' }

function allLayersVisible(): ReadonlySet<DxfLayer> {
  return new Set(DXF_LAYERS)
}

export type UseDxfPlanResult = {
  fileName: string | null
  error: string | null
  polygons: DxfParseResult['polygons']
  issues: DxfValidationIssue[]
  hasPlan: boolean
  visibleLayers: ReadonlySet<DxfLayer>
  onVisibleLayersChange: (layers: ReadonlySet<DxfLayer>) => void
  onParsed: (result: DxfParseResult, fileName: string) => void
  onError: (message: string) => void
  onCleared: () => void
  reset: () => void
}

export function useDxfPlan(): UseDxfPlanResult {
  const [state, setState] = useState<DxfPlanState>(EMPTY_STATE)
  const [visibleLayers, setVisibleLayers] = useState<ReadonlySet<DxfLayer>>(allLayersVisible)

  const onParsed = useCallback((result: DxfParseResult, fileName: string) => {
    setState({ status: 'loaded', result, fileName })
  }, [])

  const onError = useCallback((message: string) => {
    setState({ status: 'error', message })
  }, [])

  const onCleared = useCallback(() => {
    setState(EMPTY_STATE)
  }, [])

  const reset = useCallback(() => {
    setState(EMPTY_STATE)
    setVisibleLayers(allLayersVisible())
  }, [])

  const polygons = state.status === 'loaded' ? state.result.polygons : []
  const issues = state.status === 'loaded' ? state.result.issues : []

  return {
    fileName: state.status === 'loaded' ? state.fileName : null,
    error: state.status === 'error' ? state.message : null,
    polygons,
    issues,
    hasPlan: polygons.length > 0,
    visibleLayers,
    onVisibleLayersChange: setVisibleLayers,
    onParsed,
    onError,
    onCleared,
    reset,
  }
}
