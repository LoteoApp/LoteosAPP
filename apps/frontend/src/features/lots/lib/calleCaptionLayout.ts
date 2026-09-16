import type { DxfPoint } from '../types'
import { toSvgPoint } from './toSvgPoint'

const HEIGHT_FILL = 0.68
const LENGTH_FILL = 0.86
const CHAR_EM = 0.62

export function calleCaptionLayout(
  vertices: DxfPoint[],
  caption: string,
): { x: number; y: number; fontSize: number; rotate: number } | null {
  if (vertices.length < 3 || caption.length === 0) {
    return null
  }

  const strip = orientedStrip(vertices)
  if (!strip) {
    return null
  }

  const fontSize = Math.min(
    strip.width * HEIGHT_FILL,
    (strip.length * LENGTH_FILL) / (caption.length * CHAR_EM),
  )
  if (!(fontSize > 0) || !Number.isFinite(fontSize)) {
    return null
  }

  const svg = toSvgPoint(strip.center)
  const svgAngle = (Math.atan2(-strip.direction.y, strip.direction.x) * 180) / Math.PI

  return {
    x: svg.x,
    y: svg.y,
    fontSize,
    rotate: uprightAngle(svgAngle),
  }
}

function uprightAngle(degrees: number): number {
  let angle = degrees
  while (angle <= -180) angle += 360
  while (angle > 180) angle -= 360
  if (angle >= 90) return angle - 180
  if (angle < -90) return angle + 180
  return angle
}

function orientedStrip(vertices: DxfPoint[]): {
  center: DxfPoint
  length: number
  width: number
  direction: DxfPoint
} | null {
  let best: {
    width: number
    length: number
    ux: number
    uy: number
    minU: number
    maxU: number
    minN: number
    maxN: number
  } | null = null

  for (let i = 0; i < vertices.length; i++) {
    const a = vertices[i]
    const b = vertices[(i + 1) % vertices.length]
    const edgeLen = Math.hypot(b.x - a.x, b.y - a.y)
    if (!(edgeLen > 0)) {
      continue
    }

    const ux = (b.x - a.x) / edgeLen
    const uy = (b.y - a.y) / edgeLen
    const nx = -uy
    const ny = ux

    let minU = Infinity
    let maxU = -Infinity
    let minN = Infinity
    let maxN = -Infinity
    for (const point of vertices) {
      const along = point.x * ux + point.y * uy
      const across = point.x * nx + point.y * ny
      if (along < minU) minU = along
      if (along > maxU) maxU = along
      if (across < minN) minN = across
      if (across > maxN) maxN = across
    }

    const width = maxN - minN
    const length = maxU - minU
    if (!(width > 0) || !(length > 0) || !Number.isFinite(width) || !Number.isFinite(length)) {
      continue
    }

    if (!best || width < best.width || (width === best.width && length > best.length)) {
      best = { width, length, ux, uy, minU, maxU, minN, maxN }
    }
  }

  if (!best) {
    return null
  }

  const midU = (best.minU + best.maxU) / 2
  const midN = (best.minN + best.maxN) / 2
  const nx = -best.uy
  const ny = best.ux

  return {
    center: {
      x: midU * best.ux + midN * nx,
      y: midU * best.uy + midN * ny,
    },
    length: best.length,
    width: best.width,
    direction: { x: best.ux, y: best.uy },
  }
}
