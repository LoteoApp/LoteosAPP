import { toSvgPoint } from './toSvgPoint'
import type { DxfPoint } from '../types'

export function polygonToSvgPath(vertices: DxfPoint[]): string {
  if (vertices.length === 0) {
    return ''
  }

  const [first, ...rest] = vertices.map(toSvgPoint)
  let path = `M${first.x} ${first.y}`
  for (const vertex of rest) {
    path += `L${vertex.x} ${vertex.y}`
  }
  return `${path}Z`
}
