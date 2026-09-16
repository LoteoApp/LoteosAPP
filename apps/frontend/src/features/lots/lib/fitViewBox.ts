import { toSvgPoint } from './toSvgPoint'
import type { DxfPolygon } from '../types'

export type SvgViewBox = {
  x: number
  y: number
  width: number
  height: number
}

export type ZoomFocus = {
  x: number
  y: number
}

const DEFAULT_SPAN = 1
const PADDING_RATIO = 0.08
const MIN_SPAN_RATIO = 1 / 50
const MAX_SPAN_RATIO = 20

const CENTER: ZoomFocus = { x: 0.5, y: 0.5 }

export function viewBoxToString(viewBox: SvgViewBox): string {
  return `${viewBox.x} ${viewBox.y} ${viewBox.width} ${viewBox.height}`
}

export function zoomViewBox(
  viewBox: SvgViewBox,
  factor: number,
  fitted: SvgViewBox,
  focus: ZoomFocus = CENTER,
): SvgViewBox {
  if (viewBox.width <= 0 || viewBox.height <= 0) {
    return fitted
  }

  const minWidth = fitted.width * MIN_SPAN_RATIO
  const maxWidth = fitted.width * MAX_SPAN_RATIO
  const width = Math.min(Math.max(viewBox.width * factor, minWidth), maxWidth)
  const height = viewBox.height * (width / viewBox.width)

  return {
    x: viewBox.x + (viewBox.width - width) * focus.x,
    y: viewBox.y + (viewBox.height - height) * focus.y,
    width,
    height,
  }
}

export function panViewBox(
  viewBox: SvgViewBox,
  dxPx: number,
  dyPx: number,
  svgWidth: number,
  svgHeight: number,
): SvgViewBox {
  if (svgWidth === 0 || svgHeight === 0) {
    return viewBox
  }

  return {
    ...viewBox,
    x: viewBox.x - (dxPx * viewBox.width) / svgWidth,
    y: viewBox.y - (dyPx * viewBox.height) / svgHeight,
  }
}

export function fitViewBox(polygons: DxfPolygon[]): SvgViewBox {
  let minX = Infinity
  let minY = Infinity
  let maxX = -Infinity
  let maxY = -Infinity

  for (const polygon of polygons) {
    for (const vertex of polygon.vertices) {
      const { x, y } = toSvgPoint(vertex)
      if (x < minX) minX = x
      if (x > maxX) maxX = x
      if (y < minY) minY = y
      if (y > maxY) maxY = y
    }
  }

  if (!Number.isFinite(minX)) {
    return { x: 0, y: 0, width: DEFAULT_SPAN, height: DEFAULT_SPAN }
  }

  const rawWidth = maxX - minX
  const rawHeight = maxY - minY
  const width = rawWidth === 0 ? DEFAULT_SPAN : rawWidth
  const height = rawHeight === 0 ? DEFAULT_SPAN : rawHeight
  const padX = width * PADDING_RATIO
  const padY = height * PADDING_RATIO

  return {
    x: minX - padX,
    y: minY - padY,
    width: width + padX * 2,
    height: height + padY * 2,
  }
}
