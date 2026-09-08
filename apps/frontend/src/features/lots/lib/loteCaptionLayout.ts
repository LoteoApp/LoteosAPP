import type { DxfPoint } from '../types'
import { toSvgPoint } from './toSvgPoint'

const FONT_SIZE_RATIO = 0.22

export function loteCaptionLayout(
  vertices: DxfPoint[],
): { x: number; y: number; fontSize: number } | null {
  if (vertices.length === 0) {
    return null
  }

  let minX = Infinity
  let minY = Infinity
  let maxX = -Infinity
  let maxY = -Infinity
  for (const { x, y } of vertices) {
    if (x < minX) minX = x
    if (y < minY) minY = y
    if (x > maxX) maxX = x
    if (y > maxY) maxY = y
  }

  const minDim = Math.min(maxX - minX, maxY - minY)
  if (!(minDim > 0) || !Number.isFinite(minDim)) {
    return null
  }

  const svg = toSvgPoint({ x: (minX + maxX) / 2, y: (minY + maxY) / 2 })
  return { x: svg.x, y: svg.y, fontSize: minDim * FONT_SIZE_RATIO }
}
