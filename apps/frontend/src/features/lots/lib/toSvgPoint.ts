import type { DxfPoint } from '../types'

// DXF measures Y upward and SVG measures it downward, so every DXF
// coordinate that reaches the viewer has to go through here.
export function toSvgPoint(point: DxfPoint): DxfPoint {
  return { x: point.x, y: -point.y }
}
