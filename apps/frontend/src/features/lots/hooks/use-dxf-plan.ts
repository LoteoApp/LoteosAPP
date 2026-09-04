import { useCallback, useState } from 'react'
import type { DxfLayer, DxfParseResult, DxfValidationIssue } from '../types'
import { useLayerVisibility } from './use-layer-visibility'

type DxfPlanState =
  | { status: 'empty' }
  | { status: 'loaded'; result: DxfParseResult; file: File }
  | { status: 'error'; message: string }

const EMPTY_STATE: DxfPlanState = { status: 'empty' }

export type UseDxfPlanResult = {
  fileName: string | null
  file: File | null
  error: string | null
  polygons: DxfParseResult['polygons']
  issues: DxfValidationIssue[]
  hasPlan: boolean
  visibleLayers: ReadonlySet<DxfLayer>
  onVisibleLayersChange: (layers: ReadonlySet<DxfLayer>) => void
  onParsed: (result: DxfParseResult, file: File) => void
  onError: (message: string) => void
  onCleared: () => void
  reset: () => void
}

export function useDxfPlan(): UseDxfPlanResult {
  const [state, setState] = useState<DxfPlanState>(EMPTY_STATE)
  const layers = useLayerVisibility()
  const resetLayers = layers.reset

  const onParsed = useCallback((result: DxfParseResult, file: File) => {
    setState({ status: 'loaded', result, file })
  }, [])

  const onError = useCallback((message: string) => {
    setState({ status: 'error', message })
  }, [])

  const onCleared = useCallback(() => {
    setState(EMPTY_STATE)
  }, [])

  const reset = useCallback(() => {
    setState(EMPTY_STATE)
    resetLayers()
  }, [resetLayers])

  const polygons = state.status === 'loaded' ? state.result.polygons : []
  const issues = state.status === 'loaded' ? state.result.issues : []

  return {
    fileName: state.status === 'loaded' ? state.file.name : null,
    file: state.status === 'loaded' ? state.file : null,
    error: state.status === 'error' ? state.message : null,
    polygons,
    issues,
    hasPlan: polygons.length > 0,
    visibleLayers: layers.visibleLayers,
    onVisibleLayersChange: layers.onVisibleLayersChange,
    onParsed,
    onError,
    onCleared,
    reset,
  }
}
