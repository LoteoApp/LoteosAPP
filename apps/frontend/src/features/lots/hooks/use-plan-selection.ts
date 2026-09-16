import { useCallback, useState } from 'react'
import { entityEquals, type DxfPolygon, type PlanEntityRef } from '../types'

export type UsePlanSelection = {
  selectedPolygonId: string | null
  selected: PlanEntityRef | null
  select: (polygonId: string | null) => void
  selectEntity: (entity: PlanEntityRef) => void
  clear: () => void
}

export function usePlanSelection(polygons: DxfPolygon[]): UsePlanSelection {
  const [selectedPolygonId, setSelectedPolygonId] = useState<string | null>(null)
  const [selected, setSelected] = useState<PlanEntityRef | null>(null)

  const selectedPolygon =
    selectedPolygonId === null
      ? undefined
      : polygons.find((polygon) => polygon.id === selectedPolygonId)

  if (selectedPolygonId !== null && selectedPolygon === undefined) {
    setSelectedPolygonId(null)
    setSelected(null)
  }

  const select = useCallback(
    (polygonId: string | null) => {
      if (polygonId === null) {
        setSelectedPolygonId(null)
        setSelected(null)
        return
      }

      const polygon = polygons.find((item) => item.id === polygonId)
      setSelectedPolygonId(polygonId)
      setSelected(polygon?.entity ?? null)
    },
    [polygons],
  )

  const selectEntity = useCallback(
    (entity: PlanEntityRef) => {
      const polygon = polygons.find(
        (item) => item.entity !== undefined && entityEquals(item.entity, entity),
      )
      setSelectedPolygonId(polygon?.id ?? null)
      setSelected(entity)
    },
    [polygons],
  )

  const clear = useCallback(() => {
    setSelectedPolygonId(null)
    setSelected(null)
  }, [])

  return {
    selectedPolygonId: selectedPolygon === undefined ? null : selectedPolygonId,
    selected: selectedPolygon === undefined && selectedPolygonId !== null ? null : selected,
    select,
    selectEntity,
    clear,
  }
}
